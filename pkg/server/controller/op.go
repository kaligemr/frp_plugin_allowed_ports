package controller

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/gin-gonic/gin"
)

type OpController struct {
	ports map[string][]string
}

func NewOpController(ports map[string][]string) *OpController {
	return &OpController{
		ports: ports,
	}
}

func (c *OpController) Register(engine *gin.Engine) {
	engine.POST("/handler", MakeGinHandlerFunc(c.HandleLogin))
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// matchPort 检查给定端口号是否匹配规则。
// 支持三种格式：
//   - "all"：等价于 "1-65535"，匹配所有有效端口
//   - "start-end"：端口范围（如 "1-65535"、"8000-9000"）
//   - 纯数字：精确匹配单个端口
func matchPort(rule string, port int) bool {
	rule = strings.TrimSpace(strings.ToLower(rule))
	fmt.Println("[调试]] 规则:%s\t, 端口:%s\t",rule ,port )

	// all 等价于 1-65535
	if rule == "all" {
		return port >= 1 && port <= 65535
	}

	// 范围格式 start-end
	if strings.Contains(rule, "-") {
		parts := strings.SplitN(rule, "-", 2)
		if len(parts) != 2 {
			return false
		}
		start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil {
			return false
		}
		if start > end {
			start, end = end, start
		}
		return port >= start && port <= end
	}

	// 精确匹配单个端口
	p, err := strconv.Atoi(rule)
	if err != nil {
		return false
	}
	fmt.Println("调试: matchPort 返回 p: %s\t", p)
	return p == port
}

// matchDomain 检查给定完整域名是否匹配规则。
// 支持四种格式：
//   - "all"：匹配任意域名
//   - "**.domain.com"：匹配 domain.com 的所有子域名（含多级，如 a.b.c.domain.com），不匹配裸域名本身
//   - "*.domain.com"：仅匹配 domain.com 的一级子域名（如 a.domain.com），不匹配裸域名，也不匹配多级子域名
//   - 普通字符串：精确匹配
func matchDomain(rule string, domain string) bool {
	rule = strings.TrimSpace(strings.ToLower(rule))
	domain = strings.ToLower(strings.TrimSpace(domain))

	// all 匹配任意域名
	if rule == "all" {
	    fmt.Println("[调试] 允许域名 - all规则 ")
		return true
	}

	// **.domain.com —— 匹配所有子域名（一级 + 多级），不匹配裸域名
	if strings.HasPrefix(rule, "**.") {
		suffix := rule[3:] // "domain.com"
		if suffix == "" {
			return false
		}
		// 必须以 ".domain.com" 结尾，且去掉后缀后前面还有内容（即至少有一个子域标签）
		if !strings.HasSuffix(domain, "."+suffix) {
			return false
		}
		return domain != suffix
	}

	// *.domain.com —— 仅匹配一级子域名
	if strings.HasPrefix(rule, "*.") {
		suffix := rule[2:] // "domain.com"
		if suffix == "" {
			return false
		}
		dotSuffix := "." + suffix
		if !strings.HasSuffix(domain, dotSuffix) {
			return false
		}
		// 不匹配裸域名
		if domain == suffix {
			return false
		}
		// 取去掉 ".suffix" 后的前缀，其中不能再包含点（确保只有一级）
		prefix := domain[:len(domain)-len(dotSuffix)]
		return prefix != "" && !strings.Contains(prefix, ".")
	}

	// 精确匹配
	return rule == domain
}

// matchCustomDomains 检查规则是否匹配 custom domains 列表中的任意一个。
func matchCustomDomains(rule string, domains []string) bool {
	for _, d := range domains {
		if matchDomain(rule, d) {
			return true
		}
	}
	return false
}

func (c *OpController) HandleLogin(ctx *gin.Context) (interface{}, error) {
	var r plugin.Request
	var content plugin.NewProxyContent
	var res plugin.Response

	r.Content = &content
	if err := ctx.BindJSON(&r); err != nil {
		return nil, &HTTPError{
			Code: http.StatusBadRequest,
			Err:  err,
		}
	}

	fmt.Println("-------------插件: Allowed Ports--------------------")
	fmt.Printf("代理名称: %s\t代理方式%s\t", content.ProxyName, content.ProxyType)
	if strings.ToLower(content.ProxyType) == "tcp" || strings.ToLower(content.ProxyType) == "udp" {
		fmt.Printf("远程端口: %d\r\n", content.RemotePort)
	} else if strings.HasPrefix(content.ProxyType, "http") {
		fmt.Printf("自定义域名%s\r\n", content.CustomDomains)
	} else {
		fmt.Println("此类型将不进行验证")
		res.Unchange = true
		return res, nil
	}

	subdomain := content.SubDomain
	remoteport := content.RemotePort
	username := content.User.User

	if subdomain == "" && remoteport == 0 && len(content.CustomDomains) == 0 {
		res.Reject = true
		res.RejectReason = "因客户端配置错误而被拒绝"
	}

	find := false
	isTCPUDP := strings.ToLower(content.ProxyType) == "tcp" || strings.ToLower(content.ProxyType) == "udp"

	// ===== 诊断信息 =====
	rules := c.ports[username]
	fmt.Printf("[调试] 用户名: %q, 规则: %v (len=%d), 远程端口: %d, 子域: %q, 自定义域名: %v\n",
		username, rules, len(rules), remoteport, subdomain, content.CustomDomains)
	// ====================

	for _, rule := range c.ports[username] {
		// all 直接放行一切（所有协议、端口、域名）
		if strings.TrimSpace(strings.ToLower(rule)) == "all" {
			find = true
			break
		}

		// tcp/udp：端口匹配（精确 / 范围）
		if isTCPUDP && matchPort(rule, remoteport) {
			find = true
		}

		// 子域名前缀精确匹配（如规则 "foo" 匹配 subdomain="foo"）
		if subdomain != "" && strings.TrimSpace(rule) == subdomain {
			find = true
		}

		// 自定义域名匹配（精确 / *.domain.com / **.domain.com）
		if matchCustomDomains(rule, content.CustomDomains) {
			find = true
		}
	}

	if !find {
		res.Reject = true
		res.RejectReason = "客户端被禁止 => 端口或子域名无效"
	}

	if !res.Reject {
		res.Unchange = true
	}

	return res, nil
}

// ParsePorts 从 reader 中解析 ports 配置文件。
// 每行格式为 "user=rule"，支持：
//   - 以 # 开头的整行注释（跳过）
//   - 空行（跳过）
//   - 行首尾空白自动忽略
//
// 返回值：用户名 -> 规则列表。
func ParsePorts(r io.Reader) (map[string][]string, error) {
	result := make(map[string][]string)
	scanner := bufio.NewScanner(r)

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

		user := strings.TrimSpace(line[:idx])
		rule := strings.TrimSpace(line[idx+1:])

		if user == "" || rule == "" {
			continue
		}

		result[user] = append(result[user], rule)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ParsePortsFile 从指定路径读取并解析 ports 配置文件。
func ParsePortsFile(path string) (map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParsePorts(f)
}
