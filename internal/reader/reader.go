package reader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"valkey-ftdcstat/internal/discovery"
	"valkey-ftdcstat/internal/flatten"
	"valkey-ftdcstat/internal/model"
)

const magic = "VKFTDC1"

type StreamOptions struct {
	TimeRange model.TimeRange
	Flatten   flatten.Options
}

type SampleSink func(model.MetricSample) error

func ReadCapture(path string) (model.Capture, error) {
	files, warnings, err := discovery.Discover(path)
	if err != nil {
		return model.Capture{}, err
	}
	metadata, metaWarnings, err := ReadMetadata(path, files)
	if err != nil {
		return model.Capture{}, err
	}
	warnings = append(warnings, metaWarnings...)

	var samples []model.MetricSample
	sampleCount, streamWarnings, err := StreamSamples(files, StreamOptions{}, func(sample model.MetricSample) error {
		samples = append(samples, sample)
		return nil
	})
	if err != nil {
		return model.Capture{}, err
	}
	warnings = append(warnings, streamWarnings...)
	_ = sampleCount

	return model.Capture{
		Path:          path,
		Files:         filePaths(files),
		MetricSamples: samples,
		Metadata:      metadata,
		Warnings:      warnings,
	}, nil
}

func ReadMetadata(path string, files []discovery.MetricFile) (model.Metadata, []model.Warning, error) {
	metadata := model.Metadata{
		Path:  path,
		Files: filePaths(files),
	}
	var warnings []model.Warning

	for _, sidecar := range discovery.SidecarFiles(files) {
		var doc sidecarDocument
		if err := readJSONFile(sidecar.Path, &doc); err != nil {
			warnings = append(warnings, model.Warning{Source: sidecar.Path, Message: err.Error()})
			continue
		}
		metadata.Sidecars = append(metadata.Sidecars, model.SidecarMeta{
			Path:        sidecar.Path,
			CurrentFile: doc.CurrentFile,
			CreatedAtMS: doc.CreatedAtMS,
		})
	}

	metricFiles := discovery.MetricFiles(files)
	if len(metricFiles) == 0 {
		return metadata, warnings, errors.New("no metrics files found")
	}

	fileMeta, err := readMetricsFileMetadata(metricFiles[0].Path)
	if err != nil {
		warnings = append(warnings, model.Warning{Source: metricFiles[0].Path, Message: err.Error()})
	} else {
		metadata.FormatVersion = fileMeta.FormatVersion
		metadata.Module = fileMeta.Module
		metadata.Config = fileMeta.Config
	}

	first, err := readFirstSample(metricFiles[0].Path)
	if err != nil {
		warnings = append(warnings, model.Warning{Source: metricFiles[0].Path, Message: err.Error()})
		return metadata, warnings, nil
	}
	flat := flatten.Sample(first, metricFiles[0].Path, 0)
	metadata.Server = map[string]any{
		"valkey_version": firstText(flat, "valkey.info.server.valkey_version"),
		"redis_version":  firstText(flat, "valkey.info.server.redis_version"),
		"server_mode":    firstText(flat, "valkey.info.server.server_mode"),
		"process_id":     firstValkey(flat, "valkey.info.server.process_id"),
		"run_id":         firstText(flat, "valkey.info.server.run_id"),
		"hz":             firstValkey(flat, "valkey.info.server.hz"),
	}
	if v, ok := flat.Get("valkey.info.clients.maxclients"); ok {
		metadata.MaxClients = v
	}
	return metadata, warnings, nil
}

func StreamSamples(files []discovery.MetricFile, opts StreamOptions, sink SampleSink) (int, []model.Warning, error) {
	metricFiles := discovery.FilterByTimeRange(discovery.MetricFiles(files), opts.TimeRange)
	if len(metricFiles) == 0 {
		return 0, nil, errors.New("no metrics files found")
	}

	var warnings []model.Warning
	merger := mergedSampleStream{}
	count := 0
	for _, file := range metricFiles {
		fileWarnings, err := streamMetricsFile(file, opts, func(sample model.MetricSample) error {
			if !opts.TimeRange.IsZero() && !opts.TimeRange.Contains(sample.Time) {
				return nil
			}
			count++
			return merger.Add(sample, sink)
		})
		warnings = append(warnings, fileWarnings...)
		if err != nil {
			warnings = append(warnings, model.Warning{Source: file.Path, Message: err.Error()})
		}
	}
	if err := merger.Flush(sink); err != nil {
		return count, warnings, err
	}
	return count, warnings, nil
}

func streamMetricsFile(file discovery.MetricFile, opts StreamOptions, emit func(model.MetricSample) error) ([]model.Warning, error) {
	f, err := os.Open(file.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	line, err := br.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read magic %s: %w", file.Path, err)
	}
	if trimLine(line) != magic {
		return nil, fmt.Errorf("unexpected magic in %s", file.Path)
	}

	if _, err := br.ReadBytes('\n'); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read metadata %s: %w", file.Path, err)
	}

	scanner := bufio.NewScanner(br)
	scanner.Buffer(make([]byte, 1024), 16*1024*1024)
	var warnings []model.Warning
	index := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if !opts.TimeRange.IsZero() {
			if ts, ok := sampleTimestamp(line); ok && !opts.TimeRange.Contains(ts) {
				continue
			}
		}
		var sample model.Sample
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.UseNumber()
		if err := dec.Decode(&sample); err != nil {
			return warnings, fmt.Errorf("decode sample %s: %w", file.Path, err)
		}
		flat := flatten.SampleWithOptions(sample, file.Path, index, opts.Flatten)
		index++
		if flat.Time.IsZero() {
			warnings = append(warnings, model.Warning{Source: file.Path, Message: "sample without timestamp skipped"})
			continue
		}
		if err := emit(flat); err != nil {
			return warnings, err
		}
	}
	if err := scanner.Err(); err != nil {
		return warnings, fmt.Errorf("scan %s: %w", file.Path, err)
	}
	return warnings, nil
}

type fileMetadata struct {
	FormatVersion int            `json:"format_version"`
	Module        string         `json:"module"`
	Config        map[string]any `json:"config"`
}

type sidecarDocument struct {
	CurrentFile string `json:"current_file"`
	CreatedAtMS int64  `json:"created_at_ms"`
}

func readMetricsFileMetadata(path string) (fileMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return fileMetadata{}, err
	}
	defer f.Close()
	br := bufio.NewReader(f)
	line, err := br.ReadString('\n')
	if err != nil {
		return fileMetadata{}, err
	}
	if trimLine(line) != magic {
		return fileMetadata{}, fmt.Errorf("unexpected magic")
	}
	metaLine, err := br.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fileMetadata{}, err
	}
	var meta fileMetadata
	if len(bytes.TrimSpace(metaLine)) == 0 {
		return fileMetadata{}, nil
	}
	if err := json.Unmarshal(bytes.TrimSpace(metaLine), &meta); err != nil {
		return fileMetadata{}, err
	}
	return meta, nil
}

func readFirstSample(path string) (model.Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return model.Sample{}, err
	}
	defer f.Close()
	br := bufio.NewReader(f)
	if _, err := br.ReadString('\n'); err != nil {
		return model.Sample{}, err
	}
	if _, err := br.ReadBytes('\n'); err != nil && !errors.Is(err, io.EOF) {
		return model.Sample{}, err
	}
	line, err := br.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return model.Sample{}, err
	}
	var sample model.Sample
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(line)))
	dec.UseNumber()
	if err := dec.Decode(&sample); err != nil {
		return model.Sample{}, err
	}
	return sample, nil
}

func readJSONFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes.TrimSpace(data), out)
}

type mergedSampleStream struct {
	pending model.MetricSample
	have    bool
}

func (m *mergedSampleStream) Add(sample model.MetricSample, sink SampleSink) error {
	if sample.Time.IsZero() {
		return nil
	}
	if !m.have {
		m.pending = sample
		m.have = true
		return nil
	}
	if sample.Time.Equal(m.pending.Time) {
		if sample.SourceIndex >= m.pending.SourceIndex {
			m.pending = sample
		}
		return nil
	}
	if sample.Time.Before(m.pending.Time) {
		return fmt.Errorf("samples out of order: %s before %s", sample.Time.Format("2006-01-02T15:04:05Z"), m.pending.Time.Format("2006-01-02T15:04:05Z"))
	}
	if err := sink(m.pending); err != nil {
		return err
	}
	m.pending = sample
	return nil
}

func (m *mergedSampleStream) Flush(sink SampleSink) error {
	if !m.have {
		return nil
	}
	if err := sink(m.pending); err != nil {
		return err
	}
	m.have = false
	return nil
}

func filePaths(files []discovery.MetricFile) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Path)
	}
	sort.Strings(out)
	return out
}

func trimLine(s string) string {
	return string(bytes.TrimSpace([]byte(s)))
}

func firstText(flat model.MetricSample, path string) string {
	if v := flat.GetText(path); v != "" {
		return v
	}
	return flat.Text[path]
}

func firstValkey(flat model.MetricSample, path string) any {
	if v, ok := flat.Get(path); ok {
		return v
	}
	return nil
}
