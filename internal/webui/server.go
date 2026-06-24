package webui

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"valkey-ftdcstat/internal/aggregate"
	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/model"
	"valkey-ftdcstat/internal/render"
)

//go:embed static/index.html
//go:embed static/*
var assets embed.FS

type Options struct {
	View         string
	Avg          time.Duration
	RowsAveraged bool
	TimeRange    model.TimeRange
	TimeLocation *time.Location
}

type Dataset struct {
	Metadata MetadataResponse `json:"metadata"`
	Data     DataResponse     `json:"data"`
}

type MetadataResponse struct {
	View       string          `json:"view"`
	Avg        AvgInfo         `json:"avg"`
	TimeRange  TimeRangeInfo   `json:"timeRange"`
	HeaderText string          `json:"headerText"`
	Metadata   map[string]any  `json:"metadata"`
	Warnings   []model.Warning `json:"warnings,omitempty"`
	Sections   []Section       `json:"sections"`
	RowCount   int             `json:"rowCount"`
}

type DataResponse struct {
	View     string              `json:"view"`
	Avg      AvgInfo             `json:"avg"`
	Sections map[string][]string `json:"sections"`
	Rows     []DataRow           `json:"rows"`
}

type AvgInfo struct {
	Enabled  bool   `json:"enabled"`
	Bucket   string `json:"bucket,omitempty"`
	Datetime string `json:"datetime,omitempty"`
}

type TimeRangeInfo struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

type Section struct {
	Name    string   `json:"name"`
	Metrics []Metric `json:"metrics"`
}

type Metric struct {
	Column   string `json:"column"`
	Label    string `json:"label"`
	JSONName string `json:"jsonName"`
	Format   string `json:"format"`
	Default  bool   `json:"default"`
}

type DataRow struct {
	Datetime string                    `json:"datetime"`
	Sections map[string]map[string]any `json:"-"`
	Values   map[string]map[string]any `json:"-"`
}

type Server struct {
	dataset    Dataset
	indexHTML  []byte
	appJS      []byte
	styleCSS   []byte
	metaJSON   []byte
	dataJSON   []byte
	listenerFD int
	host       string
	port       int
}

func ResolveListenAddress(listen string) string {
	if strings.TrimSpace(listen) == "" {
		return "127.0.0.1:0"
	}
	return listen
}

func BuildDataset(report derive.Report, warnings []model.Warning, opts Options) Dataset {
	loc := opts.TimeLocation
	if loc == nil {
		loc = time.UTC
	}
	rows := report.Rows
	if opts.Avg > 0 && !opts.RowsAveraged {
		rows = aggregate.AverageRows(rows, opts.Avg)
	}
	sections := buildSections(report.View, report.Columns)
	return Dataset{
		Metadata: MetadataResponse{
			View:       report.View,
			Avg:        avgInfo(opts.Avg),
			TimeRange:  timeRangeInfo(opts.TimeRange, loc),
			HeaderText: render.HeaderText(report.Header),
			Metadata:   report.Metadata.Summary(),
			Warnings:   append([]model.Warning(nil), warnings...),
			Sections:   sections,
			RowCount:   len(rows),
		},
		Data: DataResponse{
			View:     report.View,
			Avg:      avgInfo(opts.Avg),
			Sections: sectionColumns(sections),
			Rows:     buildRows(rows, sections, loc),
		},
	}
}

func NewServer(dataset Dataset) (*Server, error) {
	indexBytes, err := assets.ReadFile("static/index.html")
	if err != nil {
		return nil, err
	}
	appJS, err := assets.ReadFile("static/app.js")
	if err != nil {
		return nil, err
	}
	styleCSS, err := assets.ReadFile("static/style.css")
	if err != nil {
		return nil, err
	}
	indexHTML := bytes.ReplaceAll(indexBytes, []byte("{{ .Title }}"), []byte(fmt.Sprintf("valkey-ftdcstat web UI - %s", dataset.Metadata.View)))
	metaJSON, err := marshalJSON(dataset.Metadata)
	if err != nil {
		return nil, err
	}
	dataJSON, err := marshalJSON(dataset.Data)
	if err != nil {
		return nil, err
	}
	return &Server{
		dataset:   dataset,
		indexHTML: indexHTML,
		appJS:     appJS,
		styleCSS:  styleCSS,
		metaJSON:  metaJSON,
		dataJSON:  dataJSON,
	}, nil
}

func (s *Server) Listen(listen string) (string, error) {
	host, port, err := parseListenAddress(ResolveListenAddress(listen))
	if err != nil {
		return "", err
	}
	fd, actualPort, err := listenTCP4(host, port)
	if err != nil {
		return "", err
	}
	s.listenerFD = fd
	s.host = host
	s.port = actualPort
	return fmt.Sprintf("http://%s:%d", host, actualPort), nil
}

func (s *Server) Serve() error {
	return s.serveLoop()
}

func (s *Server) Close() error {
	if s.listenerFD == 0 {
		return nil
	}
	err := syscall.Close(s.listenerFD)
	s.listenerFD = 0
	return err
}

func (s *Server) serveLoop() error {
	if s.listenerFD == 0 {
		return fmt.Errorf("server is not listening")
	}
	for {
		connFD, _, err := syscall.Accept(s.listenerFD)
		if err != nil {
			if err == syscall.EINVAL || err == syscall.EBADF {
				return nil
			}
			continue
		}
		go s.serveConn(connFD)
	}
}

func (s *Server) serveConn(fd int) {
	file := os.NewFile(uintptr(fd), "webui-conn")
	if file == nil {
		_ = syscall.Close(fd)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	method, path, err := readRequestLine(reader)
	if err != nil {
		_, _ = file.Write(httpResponse(400, "text/plain; charset=utf-8", []byte("bad request\n")))
		return
	}
	if err := discardHeaders(reader); err != nil {
		_, _ = file.Write(httpResponse(400, "text/plain; charset=utf-8", []byte("bad request\n")))
		return
	}
	if method != "GET" {
		_, _ = file.Write(httpResponse(405, "text/plain; charset=utf-8", []byte("method not allowed\n")))
		return
	}
	body, contentType, status := s.route(path)
	_, _ = file.Write(httpResponse(status, contentType, body))
}

func (s *Server) route(path string) ([]byte, string, int) {
	switch path {
	case "/":
		return s.indexHTML, "text/html; charset=utf-8", 200
	case "/app.js":
		return s.appJS, "application/javascript; charset=utf-8", 200
	case "/style.css":
		return s.styleCSS, "text/css; charset=utf-8", 200
	case "/api/metadata":
		return s.metaJSON, "application/json; charset=utf-8", 200
	case "/api/data":
		return s.dataJSON, "application/json; charset=utf-8", 200
	default:
		return []byte("not found\n"), "text/plain; charset=utf-8", 404
	}
}

func buildSections(view string, columns []string) []Section {
	metrics := make([]Metric, 0, len(columns))
	for _, column := range columns {
		if column == "time" {
			continue
		}
		metrics = append(metrics, Metric{
			Column:   column,
			Label:    column,
			JSONName: column,
			Format:   "number",
			Default:  defaultMetric(view, column),
		})
	}
	if len(metrics) == 0 {
		return nil
	}
	return []Section{{Name: view, Metrics: metrics}}
}

func defaultMetric(view, column string) bool {
	if view != "summary" {
		return true
	}
	switch column {
	case "ops/s", "hit%", "memMB", "cli", "load1", "inKB/s", "outKB/s", "repl":
		return true
	default:
		return false
	}
}

func buildRows(rows []derive.Row, sections []Section, loc *time.Location) []DataRow {
	out := make([]DataRow, 0, len(rows))
	for _, row := range rows {
		item := DataRow{
			Datetime: row.Time.In(loc).Format(time.RFC3339),
			Sections: map[string]map[string]any{},
		}
		for _, section := range sections {
			values := map[string]any{}
			for _, metric := range section.Metrics {
				values[metric.JSONName] = nil
				if value, ok := row.Values[metric.Column]; ok {
					values[metric.JSONName] = value
				}
			}
			item.Sections[section.Name] = values
		}
		item.Values = item.Sections
		out = append(out, item)
	}
	return out
}

func (r DataRow) MarshalJSON() ([]byte, error) {
	item := map[string]any{"datetime": r.Datetime}
	for key, value := range r.Sections {
		item[key] = value
	}
	return json.Marshal(item)
}

func sectionColumns(sections []Section) map[string][]string {
	out := make(map[string][]string, len(sections))
	for _, section := range sections {
		cols := make([]string, 0, len(section.Metrics))
		for _, metric := range section.Metrics {
			cols = append(cols, metric.JSONName)
		}
		sort.Strings(cols)
		out[section.Name] = cols
	}
	return out
}

func avgInfo(bucket time.Duration) AvgInfo {
	if bucket <= 0 {
		return AvgInfo{}
	}
	return AvgInfo{Enabled: true, Bucket: render.FormatAvgBucket(bucket), Datetime: "bucket_start"}
}

func timeRangeInfo(r model.TimeRange, loc *time.Location) TimeRangeInfo {
	var out TimeRangeInfo
	if !r.From.IsZero() {
		out.From = r.From.In(loc).Format(time.RFC3339)
	}
	if !r.To.IsZero() {
		out.To = r.To.In(loc).Format(time.RFC3339)
	}
	return out
}

func marshalJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func httpResponse(status int, contentType string, body []byte) []byte {
	statusText := "OK"
	switch status {
	case 200:
		statusText = "OK"
	case 400:
		statusText = "Bad Request"
	case 404:
		statusText = "Not Found"
	case 405:
		statusText = "Method Not Allowed"
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "HTTP/1.1 %d %s\r\n", status, statusText)
	fmt.Fprintf(&buf, "Content-Type: %s\r\n", contentType)
	fmt.Fprintf(&buf, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(&buf, "Connection: close\r\n\r\n")
	buf.Write(body)
	return buf.Bytes()
}

func readRequestLine(reader *bufio.Reader) (string, string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	line = strings.TrimSpace(line)
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid request line")
	}
	return parts[0], parts[1], nil
}

func discardHeaders(reader *bufio.Reader) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if line == "\r\n" || line == "\n" {
			return nil
		}
	}
}

func parseListenAddress(value string) (string, int, error) {
	lastColon := strings.LastIndex(value, ":")
	if lastColon <= 0 || lastColon == len(value)-1 {
		return "", 0, fmt.Errorf("listen address must be host:port")
	}
	host := value[:lastColon]
	portText := value[lastColon+1:]
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return "", 0, fmt.Errorf("listen port must be between 0 and 65535")
	}
	switch host {
	case "localhost":
		host = "127.0.0.1"
	case "":
		host = "0.0.0.0"
	}
	if _, err := parseIPv4(host); err != nil {
		return "", 0, err
	}
	return host, port, nil
}

func parseIPv4(host string) ([4]byte, error) {
	var out [4]byte
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return out, fmt.Errorf("listen host must be an IPv4 address or localhost")
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 255 {
			return out, fmt.Errorf("listen host must be an IPv4 address or localhost")
		}
		out[i] = byte(n)
	}
	return out, nil
}

func listenTCP4(host string, port int) (int, int, error) {
	addrBytes, err := parseIPv4(host)
	if err != nil {
		return 0, 0, err
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return 0, 0, err
	}
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		_ = syscall.Close(fd)
		return 0, 0, err
	}
	addr := &syscall.SockaddrInet4{Port: port, Addr: addrBytes}
	if err := syscall.Bind(fd, addr); err != nil {
		_ = syscall.Close(fd)
		return 0, 0, err
	}
	if err := syscall.Listen(fd, 16); err != nil {
		_ = syscall.Close(fd)
		return 0, 0, err
	}
	actual, err := syscall.Getsockname(fd)
	if err != nil {
		_ = syscall.Close(fd)
		return 0, 0, err
	}
	inet, ok := actual.(*syscall.SockaddrInet4)
	if !ok {
		_ = syscall.Close(fd)
		return 0, 0, fmt.Errorf("unexpected socket type")
	}
	return fd, inet.Port, nil
}
