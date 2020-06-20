package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	iconv "github.com/djimenez/iconv-go"
	"github.com/urfave/cli"
	"github.com/wxnacy/wgo/arrays"
)

/*
 to-do:
 1. 日志记录
 2. verbose模式完善
 3. 测试用例编写
 --- 4. 首行多一个空行的问题 ---
 5. 支持windows下的换行 (\r\n)
 6. 通过struct引用的方式传参, 减少参数个数
*/

// TemplateOpeningTag 替换正文里的 {
var TemplateOpeningTag = "___TEMPLATE_OPENING_TAG___"

// TemplateClosingTag 替换正文里的 }
var TemplateClosingTag = "___TEMPLATE_CLOSING_TAG___"

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "doit-ngxformatter"
	app.Usage = "Nginx配置文件格式化工具"
	app.Author = "yongfu"
	app.Description = "Nginx配置文件格式化工具"
	app.UsageText = "./doit-ngxformatter [-b 2]"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:     "charset, c",
			Value:    "utf-8",
			Required: false,
			Usage:    "当前支持字符集: gbk, gb18030, windows-1252, utf-8",
		},
		cli.IntFlag{
			Name:  "space, s",
			Value: 4,
			Usage: "缩进的空格数, 默认缩进4个空格",
		},
		cli.BoolFlag{
			Name:     "backup, b",
			Required: false,
			Usage:    "是否备份, 默认为false, 若需要备份请传参 -b",
		},
		cli.BoolFlag{
			Name:     "verbose, v",
			Required: false,
			Usage:    "是否显示详细信息, 默认不显示详细信息",
		},
		cli.BoolFlag{
			Name:     "testing, t",
			Required: false,
			Usage:    "测试模式, 并不会真正修改文件, 只会在终端打印格式化的配置内容",
		},
	}

	app.Action = func(c *cli.Context) error {
		blankSpace := c.Int("space")
		charset := c.String("charset")
		backup := c.Bool("backup")
		verbose := c.Bool("verbose")
		testing := c.Bool("testing")

		// 检查字符集
		if !checkCharset(charset) {
			fmt.Printf("不支持的字符集!\n 终止配置文件的格式化!\n")
			return nil
		}

		if c.NArg() > 0 {
			for _, conf := range c.Args() {
				// 防止传入的文件不存在
				if IsFile(conf) {
					// 进行格式化处理
					formatConfigFile(conf, charset, blankSpace, backup, verbose, testing)
				} else {
					fmt.Printf("文件不存在: %v\n", conf)
				}
			}
		} else {
			fmt.Printf("没有传对应的参数\n")
		}
		return nil
	}
	app.Run(os.Args)
}

// checkCharset 检查是否为受支持的字符集
func checkCharset(s string) bool {
	charsetList := []string{"gbk", "gb18030", "windows-1252", "utf-8"}
	i := arrays.ContainsString(charsetList, s)
	if i == -1 {
		return false
	}
	return true
}

// Exists 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// IsDir 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// IsFile 判断所给路径是否为文件
func IsFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

// ReadAll 读取到file中，再利用ioutil将file直接读取到[]byte中, 这是最优
func ReadAll(filePth string) string {
	f, err := os.Open(filePth)
	if err != nil {
		fmt.Println("read file fail", err)
		return ""
	}
	defer f.Close()

	fd, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("read to fd fail", err)
		return ""
	}

	return string(fd)
}

// decomposeLine 分解一行的内容
// 返回值 []string  分解后的行slice
// 返回值 bool 是否有分解发生
func decomposeLine(line string) (ls []string, mFlag bool) {
	/*
		2. 碰到多于一个分号(;)时, 需要分行, 但是引号内的分号(;)不能计算
		3.  {} 的分解
	*/
	mFlag = false
	ml := strings.Split(line, "\n")

	// 左边的数据处理
	leftL := addNewLineString(ml[0])
	leftLs := strings.Split(leftL, "\n")
	if len(leftLs) > 1 {
		mFlag = true
	}

	ls = append(ls, leftLs...)

	if len(ml) > 1 {
		mFlag = true
		// 右边的剩下的slice  #及后面的内容
		rightLs := ml[1:]
		ls = append(ls, rightLs...)
	}

	return

}

func addNewLineString(content string) string {
	var result string
	inQuotes := false
	var lastC rune
	var lastQuote rune

	var c []rune
	c = []rune(content)
	cLen := len(c) - 1
	for i, k := range c {
		// 判断当前字符为引号,并且是非转义的引号  防止 "aa'bb" 这种情况的错误判断
		if (k == '"' || k == '\'') && lastC != '\\' {
			if k != lastQuote && lastQuote == 0 {
				inQuotes = true
				lastQuote = k
			} else if k == lastQuote && lastQuote != 0 {
				inQuotes = false
				lastQuote = 0
			}
		}
		if inQuotes == true {
			result += string(k)
		} else {
			if k == ';' && i != cLen {
				result += ";\n"
			} else if k == '{' && i != cLen {
				result += " {\n"
			} else if k == '}' && i != cLen && i != 0 {
				result += "\n}\n"
			} else if k == '}' && i != cLen && i == 0 {
				result += "}\n"
			} else {
				result += string(k)
			}
		}

		lastC = k
	}
	return result
}

func applyBracketTemplateTags(contents string) string {
	var result string
	inQuotes := false
	var lastC rune
	var lastQuote rune

	var c []rune
	c = []rune(contents)
	for _, k := range c {
		// 判断当前字符为引号,并且是非转义的引号  防止 "aa'bb" 这种情况的错误判断
		if (k == '"' || k == '\'') && lastC != '\\' {
			if k != lastQuote && lastQuote == 0 {
				inQuotes = true
				lastQuote = k
			} else if k == lastQuote && lastQuote != 0 {
				inQuotes = false
				lastQuote = 0
			}
		}
		if inQuotes == true {
			if k == '{' {
				result += TemplateOpeningTag
			} else if k == '}' {
				result += TemplateClosingTag
			} else {
				result += string(k)
			}
		} else {
			if k == '#' {
				result += "\n#"
			} else {
				result += string(k)
			}
		}

		lastC = k
	}
	return result
}

func reverseInQuotesStatus(status bool) bool {
	if status == true {
		return false
	}

	return true
}

func formatConfigFile(configFilePath string, charset string, blankSpace int, backup bool, verbose bool, testing bool) {
	/*
		1. 首先以正确的编码打开文件
		2. 然后以正确的编码读取文件
		3. 判断文件内容是否为空
		4. 判断是否需要备份, 若要备份, 则进行备份(以原有的编码进行备份).
			4.1 判断是否需要显示详细信息
		5. 以utf8格式转码, 然后进行文件格式化
			5.1 将格式化后的内容, 以原编码格式写入到文件.

	*/

	// 获取文件内容, 并转换为utf-8编码
	fc := ReadAll(configFilePath)
	if charset != "utf-8" {
		// 转换为utf8字符集
		fc, _ = iconv.ConvertString(fc, charset, "utf-8")
	}

	// 判断文件是否为空
	if len(fc) == 0 {
		fmt.Printf("%v是一个空文件", configFilePath)
		return
	}

	// 此方法不用关心原来的字符集是什么, 复制的文件还是原来的字符集.
	if backup {
		_, err := CopyFile(configFilePath, configFilePath+"~")
		if err != nil {
			fmt.Println(err)
			// 当出现备份错误的时候, 不再进行后面的真正格式化
			return
		}
	}

	// 具体执行配置文件格式化
	fcNew, err := formatConfigContent(fc, blankSpace, verbose)
	if err != nil {
		fmt.Println(err)
		// 当格式化出错时, 不再进行 格式化后的文件写入到文件
		return
	}

	if testing {
		fmt.Println(fcNew)
	} else {
		// 进行编码格式转换
		if charset != "utf-8" {
			fcNew, _ = iconv.ConvertString(fcNew, "utf-8", charset)
		}

		// 写入新文件
		err = writeNewConfig(configFilePath, fcNew)
		if err != nil {
			fmt.Println(err)
		}
	}

}

func formatConfigContent(fc string, blankSpace int, verbose bool) (string, error) {
	/*
		1. 将引号内的 {} 进行替换
		2. 将内容分割为行(\n)
		3. 按行给每行进行处理
		4. 处理行中的 '{' (opening bracket)
		5. 处理缩进情况
		6. 合并行
		7. 将括号的替换进行替换回来.
		8. 清理多余的空行
		9. 返回内容
	*/

	// 按行进行分割
	lines := strings.Split(fc, "\n")
	if verbose {
		fmt.Printf("\n==Split:===\n%#v\n=======\n", lines)
	}

	lines = cleanLines(lines)
	if verbose {
		fmt.Printf("\n==cleanLines:===\n%#v\n=======\n", lines)
	}

	lines = joinOpeningBracket(lines)
	if verbose {
		fmt.Printf("\n==joinOpeningBracket:===\n%#v\n=======\n", lines)
	}

	lines = performIndentation(lines, blankSpace)
	if verbose {
		fmt.Printf("\n==performIndentation:===\n%#v\n=======\n", lines)
	}

	text := strings.Join(lines, "\n")
	if verbose {
		fmt.Printf("\n==strings.Join:===\n%#v\n=======\n", text)
	}

	text = stripBracketTemplateTags(text)
	if verbose {
		fmt.Printf("\n==stripBracketTemplateTags:===\n%#v\n=======\n", text)
	}

	return text, nil
}

func stripBracketTemplateTags(content string) string {
	content = strings.ReplaceAll(content, TemplateOpeningTag, "{")
	content = strings.ReplaceAll(content, TemplateClosingTag, "}")
	return content
}

func performIndentation(lines []string, blankSpace int) []string {
	newLines := make([]string, 0, cap(lines))
	currentIndent := 0
	for _, line := range lines {
		if (!strings.HasPrefix(line, "#")) && strings.HasSuffix(line, "}") && currentIndent > 0 {
			currentIndent--
		}

		if line != "" {
			newLines = append(newLines, strings.Repeat(" ", blankSpace*currentIndent)+line)
		} else {
			newLines = append(newLines, "")
		}

		if !strings.HasPrefix(line, "#") && strings.HasSuffix(line, "{") {
			currentIndent++
		}
	}
	return newLines
}

// joinOpeningBracket 当 { 为单独一行的时候, 合并到上一行
func joinOpeningBracket(lines []string) []string {
	newLines := make([]string, 0, cap(lines))

	lastLine := ""
	for i, l := range lines {
		if lastLine != "{" {
			if (lastLine == "" && l == "") || i == 0 {
				lastLine = l
				continue
			} else if i > 0 && l == "{" {
				newLines = append(newLines, lastLine+" {")
			} else {
				newLines = append(newLines, lastLine)
			}
		}

		lastLine = l
	}
	// 把最后一行加入进来
	newLines = append(newLines, lastLine)

	return newLines
}

func cleanLines(lines []string) []string {
	cleanedLines := make([]string, 0, cap(lines))
	for _, l := range lines {
		l = stripLine(l)
		if l == "" {
			cleanedLines = append(cleanedLines, l)
		} else if strings.HasPrefix(l, "#") {
			cleanedLines = append(cleanedLines, l)
		} else {
			l = applyBracketTemplateTags(l)
			newLines, ok := decomposeLine(l)

			if ok {
				nl := make([]string, 0, cap(newLines))
				nl = cleanAgain(newLines)
				cleanedLines = append(cleanedLines, nl...)
			} else {
				cleanedLines = append(cleanedLines, l)
			}
		}
	}
	return cleanedLines
}

func cleanAgain(lines []string) []string {
	cleanedLines := make([]string, 0, cap(lines))
	for _, l := range lines {
		l = stripLine(l)
		cleanedLines = append(cleanedLines, l)
	}
	return cleanedLines
}

func stripLine(l string) string {
	l = strings.TrimSpace(l)
	if strings.HasPrefix(l, "#") {
		return l
	}

	nl := make([]string, 0, 0)
	withInQuotes := false
	re := regexp.MustCompile(`[\s]+`)
	parts := strings.Split(l, "\"")
	for _, part := range parts {
		if withInQuotes {
			nl = append(nl, part)
		} else {
			nl = append(nl, re.ReplaceAllString(part, " "))
		}
		withInQuotes = reverseInQuotesStatus(withInQuotes)
	}
	line := strings.Join(nl, "\"")
	return line
}

func writeNewConfig(Path string, content string) error {
	text := []byte(content)
	return ioutil.WriteFile(Path, text, 0644)
}

// CopyFile 复制文件
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
