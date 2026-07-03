package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

func main() {
	mode := "headless"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	headless := mode != "visible"
	fmt.Printf("Mode: %s (headless=%v)\n", mode, headless)

	pw, err := playwright.Run()
	must(err, "start playwright")

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-blink-features=AutomationControlled",
			"--disable-features=IsolateOrigins,site-per-process",
			"--disable-dev-shm-usage",
			"--no-first-run",
			"--no-default-browser-check",
		},
	})
	must(err, "launch chromium")
	defer browser.Close()
	defer pw.Stop()

	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.60 Safari/537.36",
		),
		Locale:     playwright.String("zh-TW"),
		TimezoneId: playwright.String("Asia/Taipei"),
		Viewport:   &playwright.Size{Width: 1280, Height: 720},
	})
	must(err, "new context")

	must(ctx.AddInitScript(playwright.Script{Content: playwright.String(`
Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] });
Object.defineProperty(navigator, 'languages', { get: () => ['zh-TW', 'zh', 'en-US', 'en'] });
window.chrome = { runtime: {} };
`)}), "add init script")

	page, err := ctx.NewPage()
	must(err, "new page")

	apiCh := make(chan []byte, 1)
	page.On("response", func(r playwright.Response) {
		url := r.URL()
		status := r.Status()
		if strings.Contains(url, "/search/api/jobs") && status == 200 {
			body, _ := r.Body()
			select {
			case apiCh <- body:
			default:
			}
		}
		if strings.Contains(url, "/search/api/jobs") {
			fmt.Printf("[api/jobs] %d\n", status)
		}
	})

	// 先訪問首頁，模擬正常使用者行為
	fmt.Println("Step 1: 訪問首頁...")
	_, _ = page.Goto("https://www.104.com.tw/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(30000),
	})
	time.Sleep(3 * time.Second)

	// 模擬滑鼠移動
	page.Mouse().Move(400, 300, playwright.MouseMoveOptions{})
	time.Sleep(1 * time.Second)
	page.Mouse().Move(600, 400, playwright.MouseMoveOptions{})

	// 再前往搜尋頁
	fmt.Println("Step 2: 前往搜尋頁...")
	searchURL := "https://www.104.com.tw/jobs/search/?keyword=golang&page=1&order=2&asc=0"
	_, err = page.Goto(searchURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(60000),
	})
	if err != nil {
		fmt.Println("Nav error:", err)
	} else {
		fmt.Println("Nav complete")
	}

	// 等待 API 回應
	select {
	case body := <-apiCh:
		fmt.Printf("SUCCESS: api/jobs 200, body(%d chars): %s\n", len(body), truncate(string(body), 200))
	case <-time.After(30 * time.Second):
		title, _ := page.Title()
		fmt.Println("TIMEOUT - Page title:", title)
		// 檢查 cookie
		cookies, _ := ctx.Cookies("https://www.104.com.tw")
		for _, c := range cookies {
			if c.Name == "cf_clearance" {
				fmt.Printf("cf_clearance present: %s\n", truncate(c.Value, 20))
			}
		}
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func must(err error, msg string) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", msg, "-", err)
		os.Exit(1)
	}
}
