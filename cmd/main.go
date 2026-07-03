package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yuanyu90221/104-job-scraper/internal/formatter"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
	"github.com/yuanyu90221/104-job-scraper/internal/notifier"
	"github.com/yuanyu90221/104-job-scraper/internal/search"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		keyword    string
		area       string
		days       int
		months     int
		pages      int
		format     string
		outputFile string
		lineToken  string
		lineTopN   int
	)

	cmd := &cobra.Command{
		Use:   "104-job-scraper",
		Short: "爬取 104.com.tw 最新 golang 後端工程師職缺",
		Long: `104 Job Scraper - 搜尋 104.com.tw 職缺並輸出結果，可彙整後傳送至 LINE bot

範例:
  104-job-scraper
  104-job-scraper --keyword="golang 後端工程師" --months=3 --format=table
  104-job-scraper --keyword="golang" --area=6001001000 --format=json
  104-job-scraper --keyword="golang 後端工程師" --line-token=<TOKEN> --line-top=10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(keyword, area, days, months, pages, formatter.Format(format), outputFile, lineToken, lineTopN)
		},
	}

	cmd.Flags().StringVarP(&keyword, "keyword", "k", "golang 後端工程師", "搜尋關鍵字")
	cmd.Flags().StringVarP(&area, "area", "a", "", "地區代碼 (例: 6001001000=台北市, 留空=全台灣)")
	// 104 API isnew 最大支援 30 天；超過 30 天請搭配 --months 做客戶端過濾
	cmd.Flags().IntVarP(&days, "days", "d", 30, "API 端幾天內刊登 (0=今日, 3, 7, 14, 30)")
	cmd.Flags().IntVarP(&months, "months", "m", 3, "客戶端日期過濾：保留幾個月內刊登的職缺")
	cmd.Flags().IntVarP(&pages, "pages", "p", 15, "最多爬取幾頁 (每頁 20 筆)；三個月建議 15 頁以上")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "輸出格式: table, json, csv")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "輸出到檔案 (預設輸出至 stdout)")
	cmd.Flags().StringVar(&lineToken, "line-token", "", "LINE Notify token，設定後自動傳送摘要至 LINE bot")
	cmd.Flags().IntVar(&lineTopN, "line-top", 10, "傳送到 LINE 的前 N 筆職缺數量")

	return cmd
}

func run(keyword, area string, days, months, pages int, format formatter.Format, outputFile, lineToken string, lineTopN int) error {
	params := models.SearchParams{
		Keyword: keyword,
		Area:    area,
		Days:    days,
		Order:   2, // 依刊登日期排序
		Asc:     0, // 最新優先
	}

	s, err := search.New()
	if err != nil {
		return fmt.Errorf("初始化瀏覽器: %w", err)
	}
	defer s.Close()
	jobs, err := s.Run(params, pages)
	if err != nil {
		return fmt.Errorf("搜尋失敗: %w", err)
	}

	// 客戶端日期過濾：只保留 months 個月內刊登的職缺
	if months > 0 {
		cutoff := time.Now().AddDate(0, -months, 0)
		filtered := jobs[:0]
		for _, j := range jobs {
			if t, parseErr := time.Parse("2006-01-02", j.PublishDate); parseErr == nil && t.After(cutoff) {
				filtered = append(filtered, j)
			}
		}
		jobs = filtered
	}

	if len(jobs) == 0 {
		fmt.Fprintln(os.Stderr, "找不到符合條件的職缺。")
		return nil
	}

	fmt.Fprintf(os.Stderr, "共找到 %d 筆職缺（關鍵字: %q，近 %d 個月）\n", len(jobs), keyword, months)

	// 輸出本地結果
	out := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("建立輸出檔案: %w", err)
		}
		defer f.Close()
		out = f
	}

	if err := formatter.Print(out, jobs, format); err != nil {
		return err
	}

	// 傳送摘要至 LINE bot
	if lineToken != "" {
		ln := notifier.NewLine(lineToken)
		fmt.Fprintf(os.Stderr, "傳送前 %d 筆職缺至 LINE bot...\n", lineTopN)
		if err := ln.Send(jobs, keyword, lineTopN); err != nil {
			return fmt.Errorf("LINE 通知失敗: %w", err)
		}
		fmt.Fprintln(os.Stderr, "LINE 通知已傳送。")
	}

	return nil
}
