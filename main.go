package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// Upload function
func Upload(s3Session *session.Session,
	s3Bucket string,
	fileBytes *bytes.Reader,
	fileName string,
	realFileName string,
	len int64) error {
	svc := s3.New(s3Session)
	var ct string
	ext := filepath.Ext(fileName)
	if ext == ".png" {
		ct = "image/png"
	} else if ext == ".jpg" {
		ct = "image/jpeg"
	}
	params := &s3.PutObjectInput{
		Bucket:             aws.String(s3Bucket),
		Key:                aws.String(fileName),
		Body:               fileBytes,
		ContentLength:      aws.Int64(len),
		ContentDisposition: aws.String("inline; filename=\"" + realFileName + "\""),
		ContentType:        aws.String(ct),
	}
	_, err := svc.PutObject(params)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	const (
		awsRegion = "us-east-1"
		s3Bucket  = "i.fileit.in"
	)

	// Session init:
	cfg := aws.NewConfig().WithRegion(awsRegion)
	var s3Session *session.Session
	s3Session = session.New(cfg)

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		fmt.Fprintf(os.Stderr, "Need files to upload as params.\n")
		os.Exit(1)
	}
	for _, f := range argsWithoutProg {
		fileName := filepath.Base(f)
		ext := filepath.Ext(f)
		if !(ext == ".png" || ext == ".jpg") {
			fmt.Fprintf(os.Stderr, "%v: Not image. Ignored.\n", fileName)
			continue
		}
		upFileName, err := newUUID()
		if err != nil {
			msg := fmt.Sprintf("uuid generation error: %v", err)
			panic(msg)
		}
		upFileName = upFileName + ext
		fi, err := os.Stat(f)
		if err != nil {
			msg := fmt.Sprintf("File stat error for %v: %v", f, err)
			panic(msg)
		}
		barr, err := ioutil.ReadFile(f)
		if err != nil {
			msg := fmt.Sprintf("File read error for %v: %v", f, err)
			panic(msg)
		}
		reader := bytes.NewReader(barr)
		err = Upload(s3Session, s3Bucket, reader, upFileName, fileName, fi.Size())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error uploading %v: %v.\n", f, err)
		} else {
			fmt.Printf("![%v](https://s3.amazonaws.com/%v/%v)\n", fileName, s3Bucket, upFileName)
		}
	}
}
