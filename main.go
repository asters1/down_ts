package main

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/asters1/tools"
)

var (
	URL_HOST string
	URL_PATH string
	OUTPUT   string
	wg       sync.WaitGroup
	wg_NUM   int
	wg_m     sync.Mutex //互斥锁
	ch_str   chan string
)

func W(str string, path string) {
	wg_m.Lock()
	err := os.WriteFile(path, []byte(str), 0666)
	if err != nil {
		fmt.Println("写入失败-->", err)
		return
	}
	wg_m.Unlock()
}

func Init() (string, string) {
	Ustr_isExist := false
	Ostr_isExist := false
	URL := ""
	OUTPUT = ""
	cmd_list := os.Args

	for i := 0; i < len(cmd_list); i++ {
		if len(cmd_list[i]) > 3 {

			if cmd_list[i][:3] == "-u=" {
				Ustr_isExist = true
				URL = cmd_list[i][3:]
				if !(strings.HasPrefix(URL, "http://") || strings.HasPrefix(URL, "https://")) {
					fmt.Println("URL参数有误!")
					os.Exit(1)
				}
				URL_OBJ, err := url.Parse(URL)
				if err != nil {
					fmt.Println("解析URL出错!")
					os.Exit(1)
				}
				URL_HOST = URL_OBJ.Scheme + "://" + URL_OBJ.Host
				URL_PATH = URL[:strings.LastIndex(URL, "/")] + "/"
				// fmt.Println(URL_HOST)
				// fmt.Println(URL_PATH)

			}
			if cmd_list[i][:3] == "-o=" {
				Ostr_isExist = true
				OUTPUT = cmd_list[i][3:]
				if strings.HasPrefix(OUTPUT, "./") {
					OUTPUT = OUTPUT[2:]
				}

			}

		}
	}
	if !Ustr_isExist {
		fmt.Println("缺少参数-u!")
		os.Exit(1)

	}
	if !Ostr_isExist {
		fmt.Println("缺少参数-o!")
		os.Exit(1)
	}

	return URL, OUTPUT
}
func IsExists(path string) {
	_, err := os.Stat(path)
	if err != nil {
		fmt.Println("文件夹[ " + path + " ]不存在!")
		os.MkdirAll(path, 0666)
	}
}
func paseM3u8Url(path string) string {
	result := ""
	if strings.HasPrefix(path, "http") {
		result = path
	} else if strings.HasPrefix(path, "/") {
		result = URL_HOST + path
	} else {
		result = URL_PATH + path
	}
	// fmt.Println(result)
	return result

}
func getTslist(URL string, NAME string) []string {
	m3u8_str := ""
	result := []string{}
	for {
		m := tools.RequestClient(URL, "get", "", "")
		// fmt.Println(m)

		m_list := strings.Split(m, "\n")
		j := 0
		for i := 0; i < len(m_list); i++ {
			j_str := strconv.Itoa(j)
			if strings.TrimSpace(m_list[i]) == "#EXT-X-ENDLIST" {
				m3u8_str = m3u8_str + m_list[i] + "\n"
				goto AA
			} else {
				if i == len(m_list)-1 {
					return getTslist(paseM3u8Url(result[len(result)-1]), NAME)

				}
			}
			// fmt.Println(m_list[i])
			if strings.TrimSpace(m_list[i]) == "" {
				continue
			}
			if m_list[0] != "#EXTM3U" {
				fmt.Println("m3u8第一行不是#EXTM3U,正在退出")
				os.Exit(1)
			}
			if strings.TrimSpace(m_list[i])[:1] == "#" {
				if strings.Contains(m_list[i], "METHOD=AES") {
					re_other_pattern := `AES-128,URI="(.*?)"`
					re_key_pattern := `AES-128,URI="([^"]+)"`

					re_other := regexp.MustCompile(re_other_pattern)
					re_key := regexp.MustCompile(re_key_pattern)
					key_name := tools.GetUUID() + ".key"
					m3u8_str = m3u8_str + re_other.ReplaceAllString(m_list[i], `AES-128,URI="`+key_name+`"`) + "\n"

					key_url := re_key.FindStringSubmatch(m_list[i])[1]
					fmt.Println(key_url)
					// os.Exit(1)
					key_url = paseM3u8Url(key_url)
					// fmt.Println(p + key_name + s)
					// m3u8_str = m3u8_str + p + key_name + s + "\n"
					res := tools.RequestClient(key_url, "get", "", "")
					W(res, NAME+"/"+key_name)

				} else {
					m3u8_str = m3u8_str + m_list[i] + "\n"
				}

			} else {
				m3u8_str = m3u8_str + j_str + ".ts" + "\n"
				result = append(result, paseM3u8Url(m_list[i]))
				j++
			}

		}

	}
AA:

	// fmt.Println("########")
	W(m3u8_str, NAME+"/index.m3u8")
	return result
}
func down_ts(url string, path string) {
	fmt.Println("正在下载-->" + path)
	ts_str := tools.RequestClient(url, "get", "", "")
	W(ts_str, path)
	<-ch_str
	wg.Done()

}
func ts_pool(ts_list []string) {
	ch_str = make(chan string, wg_NUM)
	for i := 0; i < len(ts_list); i++ {
		wg.Add(1)
		ch_str <- ts_list[i]
		go down_ts(ts_list[i], OUTPUT+"/"+strconv.Itoa(i)+".ts")

	}

}

func main() {
	wg_NUM = 6
	URL, OUTPUT := Init()
	IsExists(OUTPUT)
	// getTslist(URL, OUTPUT)
	ts_list := getTslist(URL, OUTPUT)
	// fmt.Println(ts_list)
	ts_pool(ts_list)
	// fmt.Println(a)
	// fmt.Println(len(b))

}
