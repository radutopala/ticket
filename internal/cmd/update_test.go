package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UpdateSuite struct {
	suite.Suite
}

func TestUpdateSuite(t *testing.T) {
	suite.Run(t, new(UpdateSuite))
}

func (s *UpdateSuite) TestExtractTarGz() {
	tests := []struct {
		name          string
		createArchive func(t *testing.T) []byte
		wantContent   string
		wantErr       string
	}{
		{
			name: "extracts tk binary from tar.gz",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)

				content := []byte("tk binary content")
				hdr := &tar.Header{
					Name:     "tk",
					Mode:     0755,
					Size:     int64(len(content)),
					Typeflag: tar.TypeReg,
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err := tw.Write(content)
				require.NoError(t, err)
				require.NoError(t, tw.Close())
				require.NoError(t, gw.Close())

				return buf.Bytes()
			},
			wantContent: "tk binary content",
		},
		{
			name: "extracts tk binary from nested path",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)

				content := []byte("nested tk binary")
				hdr := &tar.Header{
					Name:     "tk_1.0.0_darwin_arm64/tk",
					Mode:     0755,
					Size:     int64(len(content)),
					Typeflag: tar.TypeReg,
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err := tw.Write(content)
				require.NoError(t, err)
				require.NoError(t, tw.Close())
				require.NoError(t, gw.Close())

				return buf.Bytes()
			},
			wantContent: "nested tk binary",
		},
		{
			name: "returns error when tk binary not found",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)

				content := []byte("other content")
				hdr := &tar.Header{
					Name:     "other-file",
					Mode:     0644,
					Size:     int64(len(content)),
					Typeflag: tar.TypeReg,
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err := tw.Write(content)
				require.NoError(t, err)
				require.NoError(t, tw.Close())
				require.NoError(t, gw.Close())

				return buf.Bytes()
			},
			wantErr: "tk binary not found in archive",
		},
		{
			name: "skips directories",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)

				// Add directory
				dirHdr := &tar.Header{
					Name:     "tk/",
					Mode:     0755,
					Typeflag: tar.TypeDir,
				}
				require.NoError(t, tw.WriteHeader(dirHdr))

				// Add the actual binary
				content := []byte("actual binary")
				hdr := &tar.Header{
					Name:     "tk/tk",
					Mode:     0755,
					Size:     int64(len(content)),
					Typeflag: tar.TypeReg,
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err := tw.Write(content)
				require.NoError(t, err)
				require.NoError(t, tw.Close())
				require.NoError(t, gw.Close())

				return buf.Bytes()
			},
			wantContent: "actual binary",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			archive := tt.createArchive(s.T())
			var out bytes.Buffer
			err := extractTarGz(bytes.NewReader(archive), &out)

			if tt.wantErr != "" {
				require.Error(s.T(), err)
				require.Contains(s.T(), err.Error(), tt.wantErr)
			} else {
				require.NoError(s.T(), err)
				require.Equal(s.T(), tt.wantContent, out.String())
			}
		})
	}
}

func (s *UpdateSuite) TestExtractTarGzInvalidGzip() {
	var out bytes.Buffer
	err := extractTarGz(bytes.NewReader([]byte("not a gzip")), &out)
	require.Error(s.T(), err)
}

func (s *UpdateSuite) TestExtractZip() {
	tests := []struct {
		name          string
		createArchive func(t *testing.T) []byte
		wantContent   string
		wantErr       string
	}{
		{
			name: "extracts tk.exe binary from zip",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				zw := zip.NewWriter(&buf)

				w, err := zw.Create("tk.exe")
				require.NoError(t, err)
				_, err = w.Write([]byte("tk.exe binary content"))
				require.NoError(t, err)
				require.NoError(t, zw.Close())

				return buf.Bytes()
			},
			wantContent: "tk.exe binary content",
		},
		{
			name: "extracts tk.exe from nested path",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				zw := zip.NewWriter(&buf)

				w, err := zw.Create("tk_1.0.0_windows_amd64/tk.exe")
				require.NoError(t, err)
				_, err = w.Write([]byte("nested tk.exe binary"))
				require.NoError(t, err)
				require.NoError(t, zw.Close())

				return buf.Bytes()
			},
			wantContent: "nested tk.exe binary",
		},
		{
			name: "returns error when tk.exe not found",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				zw := zip.NewWriter(&buf)

				w, err := zw.Create("other-file.txt")
				require.NoError(t, err)
				_, err = w.Write([]byte("other content"))
				require.NoError(t, err)
				require.NoError(t, zw.Close())

				return buf.Bytes()
			},
			wantErr: "tk.exe binary not found in archive",
		},
		{
			name: "skips directories",
			createArchive: func(t *testing.T) []byte {
				var buf bytes.Buffer
				zw := zip.NewWriter(&buf)

				// Add directory
				_, err := zw.Create("tk/")
				require.NoError(t, err)

				// Add the actual binary
				w, err := zw.Create("tk/tk.exe")
				require.NoError(t, err)
				_, err = w.Write([]byte("actual binary"))
				require.NoError(t, err)
				require.NoError(t, zw.Close())

				return buf.Bytes()
			},
			wantContent: "actual binary",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			archive := tt.createArchive(s.T())
			var out bytes.Buffer
			err := extractZip(bytes.NewReader(archive), &out)

			if tt.wantErr != "" {
				require.Error(s.T(), err)
				require.Contains(s.T(), err.Error(), tt.wantErr)
			} else {
				require.NoError(s.T(), err)
				require.Equal(s.T(), tt.wantContent, out.String())
			}
		})
	}
}

func (s *UpdateSuite) TestExtractZipInvalidArchive() {
	var out bytes.Buffer
	err := extractZip(bytes.NewReader([]byte("not a zip")), &out)
	require.Error(s.T(), err)
}
