package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	SERVER_HOST    = "localhost"
	SERVER_PORT    = "9980"
	SERVER_TYPE    = "tcp"
	INDEX_BODY     = "Hello World!"
	NOT_FOUND_BODY = "The requested page not found"
)

func main() {
	filename := os.Args[1] // get command line first parameter
	filedirectory := filepath.Dir(filename)
	path, err := filepath.Abs(filedirectory)
	if err != nil {
		fmt.Println("Error in file path:", err.Error())
		os.Exit(1)
	}

	fmt.Println("The root directory of web server is: ", path)

	fmt.Println("Server Running...")
	server, err := net.Listen(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer server.Close()
	fmt.Println("Listening on " + SERVER_HOST + ":" + SERVER_PORT)
	fmt.Println("Waiting for client...")
	for {
		connection, err := server.Accept()
		if err != nil {
			fmt.Println("Error accepting new connection: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("A new client connected")
		go processClient(connection, path)
	}
}

func processClient(connection net.Conn, root_direcotry string) {
	defer connection.Close()
	path, err := readAndDecodeHTTPFirstLine(connection)
	if err != nil {
		fmt.Println("Error decoding first line of request:", err.Error())
		return
	}
	fmt.Printf("Got new 'GET' request for %s\n", path)
	requestHeaders, err := readAndDecodeHTTPHeaders(connection)
	host, foundHost := requestHeaders["Host"]
	if !foundHost {
		fmt.Println("Could not found 'Host' in request headers")
		return
	}
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	filePath := string(filepath.Join(root_direcotry, path))
	content, err := os.ReadFile(filePath)

	body := ""
	statusCode := 0
	statusMessage := ""
	responseHeaders := make(map[string]string)
	if path == "/" {
		statusCode = 301
		statusMessage = "Moved Permanently"
		responseHeaders["Location"] = "index.html"
	} else if err != nil {
		body = NOT_FOUND_BODY
		statusCode = 404
		statusMessage = "Not Found"

	} else {
		body = string(content)
		statusCode = 200
		statusMessage = "OK"
		responseHeaders["Content-Length"] = strconv.Itoa(len(body))
	}

	responseHeaders["Host"] = host
	response := "HTTP/1.1 " + strconv.Itoa(statusCode) + " " + statusMessage + "\r\n" + encodeHTTPHeaders(responseHeaders) + "\r\n" + body
	_, err = connection.Write([]byte(response))
	if err != nil {
		fmt.Println("Could not write response to connection:", err.Error())
	}
	fmt.Println("Sent response with status code:", statusCode)
}

func readAndDecodeHTTPFirstLine(connection net.Conn) (string, error) {
	line, err := readLineFromConnection(connection)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Could not read a new line from connection: %s", err.Error()))
	}
	partList := strings.Split(line, " ")
	if len(partList) > 3 {
		return "", errors.New("More than 3 parts in first line")
	}
	if len(partList) < 3 {
		return "", errors.New("Less than 3 parts in first line")
	}
	method := partList[0]
	if method != "GET" {
		return "", errors.New(fmt.Sprintf("Unhandled method '%s'", method))
	}
	path := partList[1]
	version := partList[2]
	if version != "HTTP/1.1" {
		return "", errors.New(fmt.Sprintf("Unhandled method: '%s'", version))
	}
	return path, nil
}

func readLineFromConnection(connection net.Conn) (string, error) {
	data := ""
	for {
		// Read one byte per loop iteration since we want to stop reading after reaching "\r\n"
		buffer := make([]byte, 1)
		_, err := connection.Read(buffer)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Could not read new data from connection: '%s'", err.Error()))
		}
		data = data + string(buffer[0])
		if strings.HasSuffix(data, "\r\n") {
			data = data[:len(data)-2] // Remove "\r\n"
			break
		}
	}
	fmt.Printf("Read a new line from connection: '%s'\n", data)
	return data, nil
}

func readAndDecodeHTTPHeaders(connection net.Conn) (map[string]string, error) {
	headers := make(map[string]string)
	for {
		line, err := readLineFromConnection(connection)
		if line == "" {
			break
		}
		if err != nil {
			return headers, err
		}
		linePartList := strings.SplitAfterN(line, ":", 2) // SplitAfterN keeps ":" after parts
		if len(linePartList) < 2 {
			return headers, errors.New(fmt.Sprintf("Less than two parts in header line '%s'", line))
		}
		key := strings.TrimSpace(linePartList[0])
		key = key[:len(key)-1] // remove ":" at the end of key
		value := strings.TrimSpace(linePartList[1])
		fmt.Printf("Decoded new header line with key '%s' and value '%s'\n", key, value)
		headers[key] = value
	}
	return headers, nil
}

func encodeHTTPHeaders(headers map[string]string) string {
	result := ""
	for key, value := range headers {
		result = result + key + ": " + value + "\r\n"
	}
	return result
}
