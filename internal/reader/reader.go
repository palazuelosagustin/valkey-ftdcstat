package reader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"valkey-ftdcstat/internal/model"
)

const magic = "VKFTDC1"

func ReadCapture(path string) (model.Capture, error) {
	files, err := discoverFiles(path)
	if err != nil {
		return model.Capture{}, err
	}
	capture := model.Capture{Path: path, Files: files}
	for _, file := range files {
		metadata, samples, err := readFile(file)
		if err != nil {
			return model.Capture{}, err
		}
		if metadata != nil {
			capture.Metadata = append(capture.Metadata, metadata)
		}
		capture.Samples = append(capture.Samples, samples...)
	}
	sort.Slice(capture.Samples, func(i, j int) bool {
		return capture.Samples[i].TsMS < capture.Samples[j].TsMS
	})
	return capture, nil
}

func discoverFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".vkftdc") {
			continue
		}
		files = append(files, filepath.Join(path, name))
	}
	sort.Strings(files)
	return files, nil
}

func readFile(path string) (any, []model.Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, nil, fmt.Errorf("read magic %s: %w", path, err)
	}
	if strings.TrimSpace(line) != magic {
		return nil, nil, fmt.Errorf("unexpected magic in %s", path)
	}

	metaLine, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, nil, fmt.Errorf("read metadata %s: %w", path, err)
	}
	var metadata any
	if len(bytes.TrimSpace(metaLine)) > 0 {
		if err := json.Unmarshal(bytes.TrimSpace(metaLine), &metadata); err != nil {
			return nil, nil, fmt.Errorf("decode metadata %s: %w", path, err)
		}
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 16*1024*1024)
	var samples []model.Sample
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var sample model.Sample
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.UseNumber()
		if err := dec.Decode(&sample); err != nil {
			return nil, nil, fmt.Errorf("decode sample %s: %w", path, err)
		}
		samples = append(samples, sample)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return metadata, samples, nil
}
