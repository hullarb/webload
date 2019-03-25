package main

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"flag"
	"fmt"
	"io"

	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	// "github.com/google/brotli/tree/master/go/cbrotli"
)

var publicReadACL = "public-read"

func main() {
	bucket := flag.String("b", "", "name of the bucket to upload")
	region := flag.String("r", "", "aws region")
	dir := flag.String("d", "", "path of the directory which contains the files to upload")
	sync := flag.Bool("s", false, "synchronize directory, all the files taht are only present in the S3 bucket will be removed")
	compression := flag.String("c", "", "compress the files, possible values gzip or br, default: no compression")
	flag.Parse()

	if *bucket == "" || *dir == "" || (*compression != "" && *compression != "gzip" && *compression != "br") {
		fmt.Printf("Usage: %s -b BUCKET_NAME -d DIRECTORY_TO_SYNC [-c gzip|br]\n", os.Args[0])
		os.Exit(1)
	}

	var files []string
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Failed to list the content of %s: %v\n", *dir, err)
		os.Exit(1)
	}

	sess, err := session.NewSession(&aws.Config{Region: region})
	if err != nil {
		fmt.Printf("Failed to create aws session: %v\n", err)
		os.Exit(1)
	}

	uploader := s3manager.NewUploader(sess)
	uploaded := map[string]struct{}{}
	for _, file := range files {
		var b io.Reader
		var err error
		b, err = os.Open(file)
		if err != nil {
			fmt.Printf("failed to open: %s: %v\n", file, err)
			os.Exit(1)
		}
		b, err = maybeCompress(b, *compression)
		if err != nil {
			fmt.Printf("failed to compress: %s: %v\n", file, err)
			os.Exit(1)
		}
		ext := filepath.Ext(file)
		key, err := filepath.Rel(*dir, file)
		if err != nil {
			panic(err)
		}
		_, err = uploader.Upload(&s3manager.UploadInput{
			ACL:             &publicReadACL,
			Body:            b,
			ContentType:     mimeTypes[ext],
			ContentEncoding: compression,
			Bucket:          bucket,
			Key:             &key,
		})
		if err != nil {
			fmt.Printf("failed to upload: %s: %v\n", file, err)
			os.Exit(1)
		}
		uploaded[key] = struct{}{}
		fmt.Printf("%s was uploaded\n", file)
	}
	if *sync {
		fmt.Println("syncing...")
		svc := s3.New(sess)
		var ct *string
		dc := 0
		for {
			lr, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: bucket, ContinuationToken: ct})
			if err != nil {
				fmt.Printf("failed to list bucket content for syncing: %v\n", err)
				os.Exit(1)
			}
			for _, c := range lr.Contents {
				if _, ok := uploaded[*c.Key]; !ok {
					fmt.Printf("removing %s from %s\n", *c.Key, *bucket)
					_, err = svc.DeleteObject(&s3.DeleteObjectInput{Bucket: bucket, Key: c.Key})
					if err != nil {
						fmt.Printf("failed to delete %s: %v\n", *c.Key, err)
						os.Exit(1)
					}
					dc++
				}

			}
			if lr.NextContinuationToken == nil {
				break
			}
			ct = lr.NextContinuationToken
		}
		fmt.Printf("syncing has finished: %d objects were deleted.", dc)
	}
}

func maybeCompress(r io.Reader, comp string) (io.Reader, error) {
	if comp == "" {
		return r, nil
	}
	b := new(bytes.Buffer)
	var w io.WriteCloser
	switch comp {
	case "gzip":
		w = gzip.NewWriter(b)
	// TOBEDONE
	case "br":
		// w = cbrotli.NewWriter(b)
	}
	_, err := io.Copy(w, r)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	return b, err
}

var mimeTypes map[string]*string

func init() {
	c := csv.NewReader(bytes.NewBufferString(mimeTypesCSV))
	c.Comma = '\t'
	c.FieldsPerRecord = 3
	rs, err := c.ReadAll()
	if err != nil {
		panic(err)
	}
	mimeTypes = make(map[string]*string)
	for _, r := range rs {
		mimeTypes[r[0]] = &r[2]
	}
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Complete_list_of_MIME_types
const mimeTypesCSV = `.aac	AAC audio	audio/aac
.abw	AbiWord document	application/x-abiword
.arc	Archive document (multiple files embedded)	application/x-freearc
.avi	AVI: Audio Video Interleave	video/x-msvideo
.azw	Amazon Kindle eBook format	application/vnd.amazon.ebook
.bin	Any kind of binary data	application/octet-stream
.bmp	Windows OS/2 Bitmap Graphics	image/bmp
.bz	BZip archive	application/x-bzip
.bz2	BZip2 archive	application/x-bzip2
.csh	C-Shell script	application/x-csh
.css	Cascading Style Sheets (CSS)	text/css
.csv	Comma-separated values (CSV)	text/csv
.doc	Microsoft Word	application/msword
.docx	Microsoft Word (OpenXML)	application/vnd.openxmlformats-officedocument.wordprocessingml.document
.eot	MS Embedded OpenType fonts	application/vnd.ms-fontobject
.epub	Electronic publication (EPUB)	application/epub+zip
.gif	Graphics Interchange Format (GIF)	image/gif
.htm	HyperText Markup Language (HTML)	text/html
.html	HyperText Markup Language (HTML)	text/html
.ico	Icon format	image/vnd.microsoft.icon
.ics	iCalendar format	text/calendar
.jar	Java Archive (JAR)	application/java-archive
.jpeg	JPEG images	image/jpeg
.jpg	JPEG images	image/jpeg
.js	JavaScript	text/javascript
.json	JSON format	application/json
.mid	Musical Instrument Digital Interface (MIDI)	audio/midi audio/x-midi
.midi	Musical Instrument Digital Interface (MIDI)	audio/midi audio/x-midi
.mjs	JavaScript module	application/javascript
.mp3	MP3 audio	audio/mpeg
.mpeg	MPEG Video	video/mpeg
.mpkg	Apple Installer Package	application/vnd.apple.installer+xml
.odp	OpenDocument presentation document	application/vnd.oasis.opendocument.presentation
.ods	OpenDocument spreadsheet document	application/vnd.oasis.opendocument.spreadsheet
.odt	OpenDocument text document	application/vnd.oasis.opendocument.text
.oga	OGG audio	audio/ogg
.ogv	OGG video	video/ogg
.ogx	OGG	application/ogg
.otf	OpenType font	font/otf
.png	Portable Network Graphics	image/png
.pdf	Adobe Portable Document Format (PDF)	application/pdf
.ppt	Microsoft PowerPoint	application/vnd.ms-powerpoint
.pptx	Microsoft PowerPoint (OpenXML)	application/vnd.openxmlformats-officedocument.presentationml.presentation
.rar	RAR archive	application/x-rar-compressed
.rtf	Rich Text Format (RTF)	application/rtf
.sh	Bourne shell script	application/x-sh
.svg	Scalable Vector Graphics (SVG)	image/svg+xml
.swf	Small web format (SWF) or Adobe Flash document	application/x-shockwave-flash
.tar	Tape Archive (TAR)	application/x-tar
.tif	Tagged Image File Format (TIFF)	image/tiff
.tiff	Tagged Image File Format (TIFF)	image/tiff
.ttf	TrueType Font	font/ttf
.txt	Text, (generally ASCII or ISO 8859-n)	text/plain
.vsd	Microsoft Visio	application/vnd.visio
.wav	Waveform Audio Format	audio/wav
.weba	WEBM audio	audio/webm
.webm	WEBM video	video/webm
.webp	WEBP image	image/webp
.woff	Web Open Font Format (WOFF)	font/woff
.woff2	Web Open Font Format (WOFF)	font/woff2
.xhtml	XHTML	application/xhtml+xml
.xls	Microsoft Excel	application/vnd.ms-excel
.xlsx	Microsoft Excel (OpenXML)	application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
.xml	XML	application/xml
.xul	XUL	application/vnd.mozilla.xul+xml
.zip	ZIP archive	application/zip
.3gp	3GPP audio/video container	video/3gpp
.3g2	3GPP2 audio/video container	video/3gpp2
.7z	7-zip archive	application/x-7z-compressed`
