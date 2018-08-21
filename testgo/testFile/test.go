package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type WriteSeekerCloser interface {
	io.Writer
	io.Seeker
	io.Closer
}
type SettingProfile struct {
	Learned bool      `json:"learned"`
	Date    time.Time `json:"date"`
}
type UserProfile struct {
	ActiveProfile string                    `json:"active_profile"`
	Profile       map[string]SettingProfile `json:"profile"`
}
type profileInfo struct {
	profileContent        *UserProfile
	loginUser             string
	userRootPath          string
	userProfilePath       string
	settingProfilePathMap map[string]map[int]string
}

const (
	ACQ int = iota
	MONITOR
	ML
	RECORD
	PROFILE_NUM
)

type DataFile struct {
	Name string `json:"full_path"`
}

type PassThru struct {
	io.Reader
	total int64 // Total # of bytes transferred
}

// Read 'overrides' the underlying io.Reader's Read method.
// This is the one that will be called by io.Copy(). We simply
// use it to keep track of byte counts and then forward the call.
func (pt *PassThru) Read(p []byte) (int, error) {
	n, err := pt.Reader.Read(p)
	pt.total += int64(n)

	if err == nil {
		fmt.Println("Read", n, "bytes for a total of", pt.total)
	}

	return n, err
}

func main() {
	var src io.Reader    // Source file/url/etc
	var dst bytes.Buffer // Destination file/buffer/etc

	// Create some random input data.
	src = bytes.NewBufferString(strings.Repeat("Some random input data", 1000))

	// Wrap it with our custom io.Reader.
	src = &PassThru{Reader: src}

	count, err := io.Copy(&dst, src)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Transferred", count, "bytes")
}
