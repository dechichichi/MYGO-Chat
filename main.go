package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"agent/api"
	"agent/config"
	"agent/philosopher"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 配置日志
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 命令行参数
	mode := flag.String("mode", "cli", "运行模式: cli(命令行) / server(API服务器) / debate(讨论模式)")
	port := flag.String("port", ":8080", "API 服务器端口")
	philosopherType := flag.String("member", "tomori", "选择成员: tomori/anon/rana/soyo/taki")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("加载配置失败")
	}

	// 创建模型
	model := config.NewChatModel(cfg)

	switch *mode {
	case "cli":
		runCLI(model, philosopher.PhilosopherType(*philosopherType))
	case "server":
		runServer(model, *port)
	case "debate":
		runDebateDemo(model)
	default:
		log.Fatal().Str("mode", *mode).Msg("未知的运行模式")
	}
}

// runCLI 运行命令行交互模式
func runCLI(model *config.ChatModel, pType philosopher.PhilosopherType) {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     MyGO!!!!! Chat                           ║")
	fmt.Println("║                   迷子でもいい v1.0                          ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  迷路也没关系，迷路也要前进。                                ║")
	fmt.Println("║  和 MyGO 的成员们聊聊天吧。                                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 创建角色
	p := philosopher.NewPhilosopher(pType, model)
	fmt.Printf("🎸 你正在与 %s 对话\n", p.Name)
	fmt.Println("输入 'quit' 退出，输入 'switch' 切换成员")
	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Println()

	// 创建情绪分析器
	emotionAnalyzer := philosopher.NewEmotionAnalyzer(model)

	// 对话历史
	var messages []config.Message
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("你: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			fmt.Println("\n再见。迷子でもいい、迷子でも進め。")
			break
		}

		if input == "switch" {
			fmt.Println("\nMyGO!!!!! 成员:")
			fmt.Println("  1. tomori - 高松灯（主唱·感性怪女生）")
			fmt.Println("  2. anon   - 千早爱音（吉他·元气优等生）")
			fmt.Println("  3. rana   - 要乐奈（鼓手·神秘古怪少女）")
			fmt.Println("  4. soyo   - 长崎素世（贝斯·温柔大姐姐）")
			fmt.Println("  5. taki   - 椎名立希（吉他·傲娇独狼）")
			fmt.Print("请输入成员名称: ")
			if scanner.Scan() {
				newType := philosopher.PhilosopherType(strings.TrimSpace(scanner.Text()))
				p = philosopher.NewPhilosopher(newType, model)
				messages = []config.Message{} // 清空历史
				fmt.Printf("\n🎭 切换到 %s\n\n", p.Name)
			}
			continue
		}

		// 分析情绪
		emotionLevel := emotionAnalyzer.Analyze(input)

		// 添加用户消息
		messages = append(messages, config.Message{
			Role:    "user",
			Content: input,
		})

		// 获取响应
		fmt.Printf("\n%s: ", p.Name)
		response, err := p.Chat(messages, emotionLevel)
		if err != nil {
			log.Error().Err(err).Msg("对话失败")
			fmt.Println("（系统错误，请重试）")
			continue
		}

		fmt.Println(response)
		fmt.Println()

		// 添加助手消息
		messages = append(messages, config.Message{
			Role:    "assistant",
			Content: response,
		})
	}
}

// runServer 运行 API 服务器
func runServer(model *config.ChatModel, port string) {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              MyGO!!!!! Chat API Server v1.0                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	server := api.NewServer(model)
	fmt.Printf("🚀 API 服务器启动于 http://localhost%s\n", port)
	fmt.Println()
	fmt.Println("可用接口:")
	fmt.Println("  POST /api/chat          - 一对一对话")
	fmt.Println("  POST /api/debate/start  - 开始辩论")
	fmt.Println("  GET  /api/philosophers  - 获取哲学家列表")
	fmt.Println("  GET  /api/health        - 健康检查")
	fmt.Println()

	if err := server.Start(port); err != nil {
		log.Fatal().Err(err).Msg("服务器启动失败")
	}
}

// runDebateDemo 运行讨论演示
func runDebateDemo(model *config.ChatModel) {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   MyGO!!!!! 乐队讨论会                       ║")
	fmt.Println("║                   Band Meeting Time                          ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  听听 MyGO 成员们的想法吧                                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 配置讨论
	debateConfig := &philosopher.DebateConfig{
		Topic:     "乐队对我们来说意味着什么？",
		ProStance: "乐队是我们表达自我、寻找归属的地方",
		ConStance: "乐队让我们学会了面对困难和成长",
		ProPhilosophers: []philosopher.PhilosopherType{
			philosopher.TakamatsuTomori,
			philosopher.ChihayaAnon,
		},
		ConPhilosophers: []philosopher.PhilosopherType{
			philosopher.ShiinaTaki,
			philosopher.NagasakiSoyo,
		},
	}

	fmt.Printf("📜 辩题: %s\n", debateConfig.Topic)
	fmt.Printf("✅ 正方: %s\n", debateConfig.ProStance)
	fmt.Printf("❌ 反方: %s\n", debateConfig.ConStance)
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════")

	// 创建辩论引擎
	engine := philosopher.NewDebateEngine(debateConfig, model)

	// 设置发言回调
	engine.SetOnSpeech(func(speaker string, content string, phase philosopher.DebatePhase) {
		phaseNames := map[philosopher.DebatePhase]string{
			philosopher.PhaseOpening:     "【开篇立论】",
			philosopher.PhaseQuestioning: "【质询交锋】",
			philosopher.PhaseFreeDebate:  "【自由辩论】",
			philosopher.PhaseClosing:     "【总结陈词】",
		}
		fmt.Printf("\n%s %s:\n", phaseNames[phase], speaker)
		fmt.Println("────────────────────────────────────────")
		fmt.Println(content)
		fmt.Println()
	})

	// 运行讨论
	fmt.Println("\n🎬 讨论开始！\n")
	result, err := engine.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("讨论失败")
	}

	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Printf("🏁 讨论结束！共 %d 轮发言\n", len(result.Records))
}
