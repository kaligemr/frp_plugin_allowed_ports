package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofrp/fp-multiuser/pkg/server"
	"github.com/spf13/cobra"
)

const version = "1.0.3"

var (
	showVersion bool
	bindAddr    string
	// tokenFile   string
	portsFile string
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "版本信息")
	rootCmd.PersistentFlags().StringVarP(&bindAddr, "bind_addr", "l", "127.0.0.1:9000", "监听地址")
	rootCmd.PersistentFlags().StringVarP(&portsFile, "ports_file", "p", "./ports", "配置文件")
}

var rootCmd = &cobra.Command{
	Use:   "frp-allowed-ports",
	Short: "frp-allowed-ports 是 frp 服务器插件，用于支持多用户自定义配置允许的端口、域名范围",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version)
			return nil
		}

		portslist, err := ParseportsFromFile(portsFile)
		if err != nil {
			log.Printf("警告：无法解析端口配置文件 %q：%v", portsFile, err)
		} else {
			log.Printf("从 %s 文件中加载了 %d 个用户",portsFile ,len(portslist) )
		}

		s, err := server.New(server.Config{
			BindAddress: bindAddr,
			Ports:       portslist,
		})
		if err != nil {
			return err
		}
		s.Run()
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ParseportsFromFile 从文件中解析 ports 配置。
// 每行格式为 "user=rule"，支持 # 注释和空行。
func ParseportsFromFile(file string) (map[string][]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string][]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和整行注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 按第一个 = 分割
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue // 没有 = 的行忽略
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		if key == "" || val == "" {
			continue
		}

		result[key] = append(result[key], val)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
