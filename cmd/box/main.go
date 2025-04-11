package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/YouEclipse/steam-box/pkg/steambox"
	"github.com/google/go-github/github"
)

func main() {
	var err error
	steamAPIKey := os.Getenv("STEAM_API_KEY")
	steamID, _ := strconv.ParseUint(os.Getenv("STEAM_ID"), 10, 64)
	appIDs := os.Getenv("APP_ID")
	appIDList := make([]uint32, 0)

	for _, appID := range strings.Split(appIDs, ",") {
		appid, err := strconv.ParseUint(appID, 10, 32)
		if err != nil {
			continue
		}
		appIDList = append(appIDList, uint32(appid))
	}

	ghToken := os.Getenv("GH_TOKEN")
	ghUsername := os.Getenv("GH_USER")
	gistID := os.Getenv("GIST_ID")

	steamOption := "ALLTIME" // options for types of games to list: RECENT (recently played games), ALLTIME <default> (playtime of games in descending order)
	if os.Getenv("STEAM_OPTION") != "" {
		steamOption = os.Getenv("STEAM_OPTION")
	}

	multiLined := false // boolean for whether hours should have their own line - YES = true, NO = false
	if os.Getenv("MULTILINE") != "" {
		lineOption := os.Getenv("MULTILINE")
		if lineOption == "YES" {
			multiLined = true
		}
	}

	updateOption := os.Getenv("UPDATE_OPTION") // options for update: GIST (Gist only), MARKDOWN (README only), GIST_AND_MARKDOWN (Gist and README)
	markdownFiles := strings.Split(os.Getenv("MARKDOWN_FILE"), ",") // 支持多个文件名，以逗号分隔

	var updateGist, updateMarkdown bool
	if updateOption == "MARKDOWN" {
		updateMarkdown = true
	} else if updateOption == "GIST_AND_MARKDOWN" {
		updateGist = true
		updateMarkdown = true
	} else {
		updateGist = true
	}

	box := steambox.NewBox(steamAPIKey, ghUsername, ghToken)

	ctx := context.Background()

	var (
		filename string
		lines    []string
	)

	if steamOption == "ALLTIME" {
		filename = "🎮 Steam playtime leaderboard"
		lines, err = box.GetPlayTime(ctx, steamID, multiLined, appIDList...)
		if err != nil {
			panic("GetPlayTime err:" + err.Error())
		}
	} else if steamOption == "RECENT" {
		filename = "🎮 Recently played Steam games"
		lines, err = box.GetRecentGames(ctx, steamID, multiLined)
		if err != nil {
			panic("GetRecentGames err:" + err.Error())
		}
	}

	if updateGist {
		gist, err := box.GetGist(ctx, gistID)
		if err != nil {
			panic("GetGist err:" + err.Error())
		}

		f := gist.Files[github.GistFilename(filename)]

		f.Content = github.String(strings.Join(lines, "\n"))
		gist.Files[github.GistFilename(filename)] = f

		err = box.UpdateGist(ctx, gistID, gist)
		if err != nil {
			panic("UpdateGist err:" + err.Error())
		}
	}

	if updateMarkdown && len(markdownFiles) > 0 {
		title := filename
		if updateGist {
			title = fmt.Sprintf(`#### <a href="https://gist.github.com/%s" target="_blank">%s</a>`, gistID, title)
		}

		content := bytes.NewBuffer(nil)
		content.WriteString(strings.Join(lines, "\n"))

		// 遍历所有 Markdown 文件并逐一更新
		for _, markdownFile := range markdownFiles {
			markdownFile = strings.TrimSpace(markdownFile) // 清理文件名中的空格
			if markdownFile == "" {
				continue // 跳过空文件名
			}

			// 确保文件所在目录存在
			dir := filepath.Dir(markdownFile)
			if dir != "." && dir != "" {
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					fmt.Printf("Error creating directory %s for file %s: %v\n", dir, markdownFile, err)
					continue
				}
			}

			// 更新 Markdown 文件
			err = box.UpdateMarkdown(ctx, title, markdownFile, content.Bytes())
			if err != nil {
				fmt.Printf("Error updating markdown file %s: %v\n", markdownFile, err)
			} else {
				fmt.Printf("Updated markdown successfully on %s\n", markdownFile)
			}
		}
	}
}
