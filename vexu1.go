/*
Coded by Vexu — Safe load-testing worker.
Reads config JSON from first line of stdin.
Outputs JSON stats on stdout every second.
*/
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	TargetURL         string `json:"targetUrl"`
	Mode              string `json:"mode"`
	Threads           int    `json:"threads"`
	RequestsPerSecond int    `json:"requestsPerSecond"`
	DurationSeconds   int    `json:"durationSeconds"`
	RequestTimeoutMs  int    `json:"requestTimeoutMs"`
	Page              string `json:"page"`
	PostFile          string `json:"postFile"`
	UseTLS            bool   `json:"useTls"`
}

var cfg Config

var (
	abcd = "asdfghjklqwertyuiopzxcvbnmASDFGHJKLQWERTYUIOPZXCVBNM"

	acceptall = []string{
		"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n",
		"Accept-Encoding: gzip, deflate\r\n",
		"Accept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n",
		"Accept: text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Charset: iso-8859-1\r\nAccept-Encoding: gzip\r\n",
		"Accept: application/xml,application/xhtml+xml,text/html;q=0.9, text/plain;q=0.8,image/png,*/*;q=0.5\r\nAccept-Charset: iso-8859-1\r\n",
	}

	choice  = []string{"Macintosh", "Windows", "X11"}
	choice2 = []string{"68K", "PPC", "Intel Mac OS X"}
	choice3 = []string{"Win3.11", "WinNT3.51", "WinNT4.0", "Windows NT 5.0", "Windows NT 5.1", "Windows NT 5.2", "Windows NT 6.0", "Windows NT 6.1", "Windows NT 6.2", "Win 9x 4.90", "WindowsCE", "Windows XP", "Windows 7", "Windows 8", "Windows NT 10.0; Win64; x64"}
	choice4 = []string{"Linux i686", "Linux x86_64"}
	choice5 = []string{"chrome", "spider", "ie"}
	choice6 = []string{".NET CLR", "SV1", "Tablet PC", "Win64; IA64", "Win64; x64", "WOW64"}

	spider = []string{
		"AdsBot-Google ( http://www.google.com/adsbot.html)",
		"Baiduspider ( http://www.baidu.com/search/spider.htm)",
		"FeedFetcher-Google; ( http://www.google.com/feedfetcher.html)",
		"Googlebot/2.1 ( http://www.googlebot.com/bot.html)",
		"Googlebot-Image/1.0",
		"Googlebot-News",
		"Googlebot-Video/1.0",
	}

	referers = []string{
		"https://www.google.com/search?q=",
		"https://check-host.net/",
		"https://www.facebook.com/",
		"https://www.youtube.com/",
		"https://www.bing.com/search?q=",
		"https://r.search.yahoo.com/",
		"https://vk.com/profile.php?auto=",
		"https://www.usatoday.com/search/results?q=",
		"https://help.baidu.com/searchResult?keywords=",
		"https://steamcommunity.com/market/search?q=",
		"https://www.ted.com/search?q=",
		"https://play.google.com/store/search?q=",
	}

	host   string
	port   string
	page   string
	mode   string
	key    string
	useTLS bool

	successCount int64
	failCount    int64
	latencySum   int64
	latencyCount int64
)

func init() { rand.Seed(time.Now().UnixNano()) }

func getuseragent() string {
	platform := choice[rand.Intn(len(choice))]
	var osStr string
	if platform == "Macintosh" {
		osStr = choice2[rand.Intn(len(choice2))]
	} else if platform == "Windows" {
		osStr = choice3[rand.Intn(len(choice3))]
	} else {
		osStr = choice4[rand.Intn(len(choice4))]
	}
	browser := choice5[rand.Intn(len(choice5))]
	if browser == "chrome" {
		webkit := strconv.Itoa(rand.Intn(599-500) + 500)
		uwu := strconv.Itoa(rand.Intn(99)) + ".0" + strconv.Itoa(rand.Intn(9999)) + "." + strconv.Itoa(rand.Intn(999))
		return "Mozilla/5.0 (" + osStr + ") AppleWebKit/" + webkit + ".0 (KHTML, like Gecko) Chrome/" + uwu + " Safari/" + webkit
	} else if browser == "ie" {
		uwu := strconv.Itoa(rand.Intn(99)) + ".0"
		engine := strconv.Itoa(rand.Intn(99)) + ".0"
		token := ""
		if rand.Intn(2) == 1 {
			token = choice6[rand.Intn(len(choice6))] + "; "
		}
		return "Mozilla/5.0 (compatible; MSIE " + uwu + "; " + osStr + "; " + token + "Trident/" + engine + ")"
	}
	return spider[rand.Intn(len(spider))]
}

func buildHeader() string {
	addr := host + ":" + port
	h := ""
	if mode == "get" {
		h += " HTTP/1.1\r\nHost: " + addr + "\r\n"
		h += "Connection: Keep-Alive\r\nCache-Control: max-age=0\r\n"
		h += "User-Agent: " + getuseragent() + "\r\n"
		h += acceptall[rand.Intn(len(acceptall))]
		h += "Referer: " + referers[rand.Intn(len(referers))] + "\r\n"
	} else {
		data := "f"
		if cfg.PostFile != "" && cfg.PostFile != "nil" {
			if b, err := os.ReadFile(cfg.PostFile); err == nil {
				data = string(b)
			}
		}
		h += "POST " + page + " HTTP/1.1\r\nHost: " + addr + "\r\n"
		h += "Connection: Keep-Alive\r\nContent-Type: application/x-www-form-urlencoded\r\n"
		h += "Content-Length: " + strconv.Itoa(len(data)) + "\r\n"
		h += "Accept-Encoding: gzip, deflate\r\n\r\n" + data + "\r\n"
	}
	return h
}

var rateTicker *time.Ticker

func flood(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	addr := host + ":" + port

	for {
		select {
		case <-ctx.Done():
			return
		case <-rateTicker.C:
		}

		start := time.Now()
		var conn net.Conn
		var err error

		if useTLS {
			dialer := &net.Dialer{Timeout: time.Duration(cfg.RequestTimeoutMs) * time.Millisecond}
			conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         host,
			})
		} else {
			conn, err = net.DialTimeout("tcp", addr, time.Duration(cfg.RequestTimeoutMs)*time.Millisecond)
		}

		if err != nil {
			atomic.AddInt64(&failCount, 1)
			continue
		}

		var req strings.Builder
		if mode == "get" {
			req.WriteString("GET " + page + key)
			req.WriteString(strconv.Itoa(rand.Intn(2147483647)))
			req.WriteString(string(abcd[rand.Intn(len(abcd))]))
			req.WriteString(string(abcd[rand.Intn(len(abcd))]))
			req.WriteString(string(abcd[rand.Intn(len(abcd))]))
			req.WriteString(string(abcd[rand.Intn(len(abcd))]))
			req.WriteString(buildHeader())
		} else {
			req.WriteString(buildHeader())
		}
		req.WriteString("\r\n")

		_ = conn.SetWriteDeadline(time.Now().Add(time.Duration(cfg.RequestTimeoutMs) * time.Millisecond))
		_, werr := conn.Write([]byte(req.String()))
		conn.Close()

		if werr != nil {
			atomic.AddInt64(&failCount, 1)
		} else {
			atomic.AddInt64(&successCount, 1)
			atomic.AddInt64(&latencySum, int64(time.Since(start).Milliseconds()))
			atomic.AddInt64(&latencyCount, 1)
		}
	}
}

func statsReporter(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s := atomic.LoadInt64(&successCount)
			f := atomic.LoadInt64(&failCount)
			ls := atomic.LoadInt64(&latencySum)
			lc := atomic.LoadInt64(&latencyCount)
			var avg int64
			if lc > 0 {
				avg = ls / lc
			}
			out, _ := json.Marshal(map[string]interface{}{
				"success":      s,
				"failed":       f,
				"avgLatencyMs": avg,
			})
			fmt.Println(string(out))
		}
	}
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		if err := json.Unmarshal([]byte(scanner.Text()), &cfg); err != nil {
			fmt.Fprintln(os.Stderr, "config parse error:", err)
			os.Exit(1)
		}
	}
	if cfg.TargetURL == "" {
		fmt.Fprintln(os.Stderr, "targetUrl required")
		os.Exit(1)
	}

	u, err := url.Parse(cfg.TargetURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad url:", err)
		os.Exit(1)
	}
	host = u.Hostname()
	port = u.Port()
	if u.Scheme == "https" {
		useTLS = true
		if port == "" {
			port = "443"
		}
	} else if port == "" {
		port = "80"
	}

	page = u.Path
	if page == "" {
		page = "/"
	}
	if cfg.Page != "" && cfg.Page != "/" {
		page = cfg.Page
	}

	mode = strings.ToLower(cfg.Mode)
	if mode != "post" {
		mode = "get"
	}

	if strings.Contains(page, "?") {
		key = "&"
	} else {
		key = "?"
	}

	if cfg.Threads < 1 {
		cfg.Threads = 1
	}
	if cfg.RequestsPerSecond < 1 {
		cfg.RequestsPerSecond = 1
	}
	if cfg.DurationSeconds < 1 {
		cfg.DurationSeconds = 60
	}
	if cfg.RequestTimeoutMs < 100 {
		cfg.RequestTimeoutMs = 5000
	}

	interval := time.Second / time.Duration(cfg.RequestsPerSecond)
	rateTicker = time.NewTicker(interval)
	defer rateTicker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DurationSeconds)*time.Second)
	defer cancel()

	fmt.Fprintf(os.Stderr, "[go-worker] target=%s mode=%s threads=%d rps=%d duration=%ds\n",
		cfg.TargetURL, mode, cfg.Threads, cfg.RequestsPerSecond, cfg.DurationSeconds)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Threads; i++ {
		wg.Add(1)
		go flood(ctx, &wg)
	}
	go statsReporter(ctx)
	wg.Wait()

	s := atomic.LoadInt64(&successCount)
	f := atomic.LoadInt64(&failCount)
	out, _ := json.Marshal(map[string]interface{}{"success": s, "failed": f, "done": true})
	fmt.Println(string(out))
}