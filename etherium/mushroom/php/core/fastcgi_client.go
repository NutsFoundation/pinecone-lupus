package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/NutsFoundation/pinecone-lupus/etherium/mushroom/php/arch/entity"
)

const (
	fcgiVersion1       = 1
	fcgiBeginRequest   = 1
	fcgiAbortRequest   = 2
	fcgiEndRequest     = 3
	fcgiParams         = 4
	fcgiStdin          = 5
	fcgiStdout         = 6
	fcgiStderr         = 7
	fcgiResponder      = 1
	fcgiRequestID      = 1
	fcgiMaxContentSize = 65535
)

type FastCGIClient struct {
	config entity.MushroomConfig
}

func NewFastCGIClient(config entity.MushroomConfig) *FastCGIClient {
	return &FastCGIClient{config: normalizeConfig(config)}
}

func (c *FastCGIClient) Execute(ctx context.Context, request *entity.MushroomRequest, target *entity.ResolvedTarget) (*entity.MushroomResponse, error) {
	if request == nil || target == nil {
		return nil, errors.New("PHP_MUSHROOM_FASTCGI_BAD_REQUEST")
	}
	ctx = nonNilContext(ctx)
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(c.config.Runtime.RequestTimeoutMillis)*time.Millisecond)
		defer cancel()
	}

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(c.config.Runtime.Host, strconv.Itoa(c.config.Runtime.Port)))
	if err != nil {
		return nil, fmt.Errorf("PHP_MUSHROOM_FASTCGI_DIAL_FAILED: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if err := writeBeginRequest(conn); err != nil {
		return nil, err
	}
	if err := writeParams(conn, buildFastCGIParams(c.config, request, target)); err != nil {
		return nil, err
	}
	if err := writeRecord(conn, fcgiParams, nil); err != nil {
		return nil, err
	}
	if err := writeStream(conn, fcgiStdin, request.Body); err != nil {
		return nil, err
	}
	if err := writeRecord(conn, fcgiStdin, nil); err != nil {
		return nil, err
	}

	stdout, stderr, err := readFastCGIResponse(conn, c.config.Runtime.MaxResponseBodyBytes)
	if err != nil {
		return nil, err
	}
	if len(stderr) > 0 {
		fmt.Printf("[PHP-Mushroom] stderr: %s\n", strings.TrimSpace(string(stderr)))
	}
	return parseCGIResponse(stdout), nil
}

func (c *FastCGIClient) ExecuteScript(
	ctx context.Context,
	request *entity.MushroomRequest,
	scriptPath string,
	documentRoot string,
	scriptName string,
) (*entity.MushroomResponse, error) {
	if request == nil {
		request = &entity.MushroomRequest{}
	}
	if strings.TrimSpace(scriptPath) == "" {
		return nil, errors.New("PHP_MUSHROOM_SCRIPT_FILENAME_REQUIRED")
	}
	if strings.TrimSpace(documentRoot) == "" {
		documentRoot = "."
	}
	if strings.TrimSpace(scriptName) == "" {
		scriptName = "/" + filepath.ToSlash(filepath.Base(scriptPath))
	}
	request = c.normalizeScriptRequest(request, scriptName)
	target := &entity.ResolvedTarget{
		DocumentRoot: documentRoot,
		ScriptPath:   scriptPath,
		ScriptName:   scriptName,
		EntryPath:    scriptName,
	}
	return c.Execute(ctx, request, target)
}

func (c *FastCGIClient) normalizeScriptRequest(request *entity.MushroomRequest, scriptName string) *entity.MushroomRequest {
	if request == nil {
		request = &entity.MushroomRequest{}
	}
	if strings.TrimSpace(request.Method) == "" {
		request.Method = http.MethodGet
	}
	if strings.TrimSpace(request.Path) == "" {
		request.Path = scriptName
	}
	if strings.TrimSpace(request.URI) == "" {
		request.URI = request.Path
	}
	if strings.TrimSpace(request.Scheme) == "" {
		request.Scheme = "http"
	}
	if strings.TrimSpace(request.Host) == "" {
		request.Host = net.JoinHostPort(c.config.Runtime.Host, strconv.Itoa(c.config.Runtime.Port))
	}
	if request.Header == nil {
		request.Header = http.Header{}
	}
	return request
}

func writeBeginRequest(writer io.Writer) error {
	body := []byte{0, fcgiResponder, 0, 0, 0, 0, 0, 0}
	return writeRecord(writer, fcgiBeginRequest, body)
}

func writeParams(writer io.Writer, params map[string]string) error {
	buffer := bytes.Buffer{}
	for name, value := range params {
		buffer.Write(encodeNameValuePair(name, value))
		if buffer.Len() > 32000 {
			if err := writeRecord(writer, fcgiParams, buffer.Bytes()); err != nil {
				return err
			}
			buffer.Reset()
		}
	}
	if buffer.Len() > 0 {
		return writeRecord(writer, fcgiParams, buffer.Bytes())
	}
	return nil
}

func writeStream(writer io.Writer, recordType byte, data []byte) error {
	for len(data) > 0 {
		chunkSize := len(data)
		if chunkSize > fcgiMaxContentSize {
			chunkSize = fcgiMaxContentSize
		}
		if err := writeRecord(writer, recordType, data[:chunkSize]); err != nil {
			return err
		}
		data = data[chunkSize:]
	}
	return nil
}

func writeRecord(writer io.Writer, recordType byte, content []byte) error {
	contentLength := len(content)
	paddingLength := (8 - (contentLength % 8)) % 8
	header := []byte{
		fcgiVersion1,
		recordType,
		0, fcgiRequestID,
		byte(contentLength >> 8), byte(contentLength),
		byte(paddingLength),
		0,
	}
	if _, err := writer.Write(header); err != nil {
		return err
	}
	if contentLength > 0 {
		if _, err := writer.Write(content); err != nil {
			return err
		}
	}
	if paddingLength > 0 {
		_, err := writer.Write(make([]byte, paddingLength))
		return err
	}
	return nil
}

func encodeNameValuePair(name, value string) []byte {
	buffer := bytes.Buffer{}
	writeLength(&buffer, len(name))
	writeLength(&buffer, len(value))
	buffer.WriteString(name)
	buffer.WriteString(value)
	return buffer.Bytes()
}

func writeLength(buffer *bytes.Buffer, length int) {
	if length < 128 {
		buffer.WriteByte(byte(length))
		return
	}
	encoded := uint32(length) | 0x80000000
	_ = binary.Write(buffer, binary.BigEndian, encoded)
}

func readFastCGIResponse(reader io.Reader, maxBytes int64) ([]byte, []byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(reader, header); err != nil {
			return nil, nil, err
		}
		recordType := header[1]
		contentLength := int(header[4])<<8 | int(header[5])
		paddingLength := int(header[6])
		content := make([]byte, contentLength)
		if contentLength > 0 {
			if _, err := io.ReadFull(reader, content); err != nil {
				return nil, nil, err
			}
		}
		if paddingLength > 0 {
			if _, err := io.CopyN(io.Discard, reader, int64(paddingLength)); err != nil {
				return nil, nil, err
			}
		}
		switch recordType {
		case fcgiStdout:
			if contentLength > 0 {
				stdout.Write(content)
				if maxBytes > 0 && int64(stdout.Len()) > maxBytes {
					return nil, nil, errors.New("PHP_MUSHROOM_RESPONSE_TOO_LARGE")
				}
			}
		case fcgiStderr:
			stderr.Write(content)
		case fcgiEndRequest:
			return stdout.Bytes(), stderr.Bytes(), nil
		case fcgiAbortRequest:
			return nil, nil, errors.New("PHP_MUSHROOM_FASTCGI_ABORTED")
		}
	}
}

func buildFastCGIParams(config entity.MushroomConfig, request *entity.MushroomRequest, target *entity.ResolvedTarget) map[string]string {
	contentLength := strconv.Itoa(len(request.Body))
	serverName := request.Host
	if strings.Contains(serverName, ":") {
		serverName, _, _ = net.SplitHostPort(serverName)
	}
	scriptName := target.ScriptName
	if strings.TrimSpace(scriptName) == "" {
		scriptName = target.EntryPath
	}
	params := map[string]string{
		"GATEWAY_INTERFACE": "CGI/1.1",
		"SERVER_SOFTWARE":   "pinecone-php-mushroom",
		"SERVER_PROTOCOL":   "HTTP/1.1",
		"REQUEST_METHOD":    request.Method,
		"DOCUMENT_ROOT":     target.DocumentRoot,
		"SCRIPT_FILENAME":   target.ScriptPath,
		"SCRIPT_NAME":       scriptName,
		"REQUEST_URI":       request.URI,
		"QUERY_STRING":      request.RawQuery,
		"CONTENT_TYPE":      request.Header.Get("Content-Type"),
		"CONTENT_LENGTH":    contentLength,
		"REMOTE_ADDR":       request.RemoteAddr,
		"SERVER_NAME":       serverName,
		"SERVER_PORT":       strconv.Itoa(config.Runtime.Port),
		"HTTPS":             "",
	}
	for key, value := range target.ExtraParams {
		params[key] = value
	}
	return params
}

func parseCGIResponse(raw []byte) *entity.MushroomResponse {
	statusCode := http.StatusOK
	header := http.Header{}
	body := raw
	if split := bytes.Index(raw, []byte("\r\n\r\n")); split >= 0 {
		headerBlock := raw[:split]
		body = raw[split+4:]
		scanner := bufio.NewScanner(bytes.NewReader(headerBlock))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := http.CanonicalHeaderKey(strings.TrimSpace(parts[0]))
			value := strings.TrimSpace(parts[1])
			if strings.EqualFold(key, "Status") {
				if fields := strings.Fields(value); len(fields) > 0 {
					if parsed, err := strconv.Atoi(fields[0]); err == nil {
						statusCode = parsed
					}
				}
				continue
			}
			header.Add(key, value)
		}
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/html; charset=utf-8")
	}
	return &entity.MushroomResponse{
		StatusCode: statusCode,
		Header:     copySafeHeaders(header),
		Body:       body,
	}
}
