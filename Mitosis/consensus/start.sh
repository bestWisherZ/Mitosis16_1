#!/bin/bash

# 设置日志文件的路径为 ../../log 目录下，并使用默认的日志文件名称 log.txt
log_file=../../log/log.txt

# 如果提供了参数，则使用提供的日志文件名称
if [ ! -z "$1" ]; then
    log_file=../../log/"$1"
fi

# 运行 go test 并将输出重定向到指定的日志文件
go test -v -timeout 3600s | tee "$log_file"