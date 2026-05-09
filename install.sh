#!/bin/bash
go mod tidy
go build -o builder ./Builder/builder.go