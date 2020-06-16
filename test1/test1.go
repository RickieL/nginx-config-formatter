package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// CopyFile 文件复制
func CopyFile(dstName, srcName string) (writeen int64, err error) {
	src, err := os.Open(dstName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer src.Close()

	dst, err := os.OpenFile(srcName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dst.Close()

	return io.Copy(dst, src)

}

func main() {
	/*
		1. 在 ;(\s*)# 情况进行分行
		2. 碰到多于一个分号(;)时, 需要分行, 但是引号内的分号(;)不能计算
		3.  {} 的分解
	*/
	// s := "sld{kfl; skd;}jfl;   \n# lskdf {jladkf;} #lsdkf; dfl;"
	// lines := strings.Split(s, "\n")

	// fmt.Printf("lines: %v\n, len: %v, cap: %v", lines, len(lines), cap(lines))

	// if len(lines) > 1 {
	// 	fmt.Printf("line[0]: %v", lines[0])
	// }

	a := "hahaha"
	b := "hehehe"
	c := strings.Join([]string{a, b}, ",")
	println(c)
}
