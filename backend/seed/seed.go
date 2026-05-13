package main

import (
	"log"

	"interview-platform/config"
	"interview-platform/database"
	"interview-platform/models"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()

	if err := database.Init(cfg.DSN()); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Println("数据库初始化成功，开始填充数据...")

	seed()
	log.Println("数据填充完成！")
}

func seed() {
	// ====== Users ======
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	users := []models.User{
		{Username: "张三", Email: "zhangsan@test.com", PasswordHash: string(hash)},
		{Username: "李四", Email: "lisi@test.com", PasswordHash: string(hash)},
		{Username: "王五", Email: "wangwu@test.com", PasswordHash: string(hash)},
	}
	for _, u := range users {
		database.DB.Create(&u)
	}
	log.Println("已创建 3 个测试用户 (密码: 123456)")

	// ====== Categories ======
	techParent := models.Category{Name: "技术类", Type: models.CategoryTech, SortOrder: 1}
	nonTechParent := models.Category{Name: "非技术类", Type: models.CategoryNonTech, SortOrder: 2}
	database.DB.Create(&techParent)
	database.DB.Create(&nonTechParent)

	techCats := []models.Category{
		{Name: "前端", ParentID: &techParent.ID, Type: models.CategoryTech, SortOrder: 1, Icon: "code"},
		{Name: "后端", ParentID: &techParent.ID, Type: models.CategoryTech, SortOrder: 2, Icon: "server"},
		{Name: "数据库", ParentID: &techParent.ID, Type: models.CategoryTech, SortOrder: 3, Icon: "database"},
		{Name: "算法", ParentID: &techParent.ID, Type: models.CategoryTech, SortOrder: 4, Icon: "algorithm"},
		{Name: "系统设计", ParentID: &techParent.ID, Type: models.CategoryTech, SortOrder: 5, Icon: "design"},
	}
	for _, c := range techCats {
		database.DB.Create(&c)
	}

	nonTechCats := []models.Category{
		{Name: "行为面试", ParentID: &nonTechParent.ID, Type: models.CategoryNonTech, SortOrder: 1, Icon: "behavior"},
		{Name: "HR 面试", ParentID: &nonTechParent.ID, Type: models.CategoryNonTech, SortOrder: 2, Icon: "hr"},
		{Name: "情景题", ParentID: &nonTechParent.ID, Type: models.CategoryNonTech, SortOrder: 3, Icon: "scenario"},
	}
	for _, c := range nonTechCats {
		database.DB.Create(&c)
	}
	log.Println("已创建分类")

	// ====== Questions ======
	questions := []models.Question{
		// 前端
		{CategoryID: techCats[0].ID, Title: "React 中 useState 和 useReducer 的区别是什么？分别在什么场景下使用？",
			Content: "请详细说明 React Hooks 中 useState 和 useReducer 的区别、各自的适用场景，以及性能方面的考量。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionTech},
		{CategoryID: techCats[0].ID, Title: "请解释浏览器从输入 URL 到页面渲染的完整过程",
			Content: "描述从用户在浏览器中输入 URL 到页面完全渲染展示的完整过程，包括 DNS 解析、TCP 连接、HTTP 请求、浏览器渲染等环节。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionTech},
		{CategoryID: techCats[0].ID, Title: "什么是闭包？实际开发中有哪些应用场景？",
			Content: "请解释 JavaScript 中闭包的概念、原理以及在实际项目中的常见应用场景，并说明闭包可能带来的内存问题。",
			Difficulty: models.DifficultyEasy, Type: models.QuestionTech},
		{CategoryID: techCats[0].ID, Title: "CSS 中 flex 和 grid 布局的区别与联系",
			Content: "请比较 CSS Flexbox 和 Grid 布局的异同点，各自适合什么样的布局场景，并给出实际案例。",
			Difficulty: models.DifficultyEasy, Type: models.QuestionTech},
		{CategoryID: techCats[0].ID, Title: "Vue 3 的 Composition API 和 Options API 如何选择？",
			Content: "Vue 3 提供了 Composition API 和 Options API 两种代码组织方式，请分析两者的优缺点和各自的适用场景。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionTech},

		// 后端
		{CategoryID: techCats[1].ID, Title: "RESTful API 设计中有哪些最佳实践？",
			Content: "请列举 RESTful API 设计的主要原则和最佳实践，包括 URL 设计、HTTP 方法选择、状态码使用、版本管理等。",
			Difficulty: models.DifficultyEasy, Type: models.QuestionTech},
		{CategoryID: techCats[1].ID, Title: "如何设计一个高并发的秒杀系统？",
			Content: "请从架构层面设计一个支持高并发的秒杀系统，包括流量控制、缓存策略、数据库优化、防超卖等关键环节的设计。",
			Difficulty: models.DifficultyHard, Type: models.QuestionTech},
		{CategoryID: techCats[1].ID, Title: "JWT 和 Session 认证方式的区别和优缺点",
			Content: "请比较 JWT（JSON Web Token）和 Session 两种用户认证方式的原理、优缺点及各自的适用场景。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionTech},

		// 数据库
		{CategoryID: techCats[2].ID, Title: "MySQL 中什么是索引？有哪些类型的索引？",
			Content: "请解释 MySQL 中索引的概念、作用，以及 B+树索引、哈希索引、全文索引等不同类型索引的原理和适用场景。",
			Difficulty: models.DifficultyEasy, Type: models.QuestionTech},
		{CategoryID: techCats[2].ID, Title: "什么是数据库事务的 ACID 特性？",
			Content: "请解释数据库事务的 ACID 四个特性（原子性、一致性、隔离性、持久性），以及隔离级别和各自解决的问题。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionTech},
		{CategoryID: techCats[2].ID, Title: "简述 Redis 的数据结构及其使用场景",
			Content: "请列举 Redis 支持的主要数据类型（String、Hash、List、Set、ZSet 等），并说明各自的实际应用场景。",
			Difficulty: models.DifficultyEasy, Type: models.QuestionTech},

		// 算法
		{CategoryID: techCats[3].ID, Title: "请实现一个 LRU 缓存淘汰算法",
			Content: "设计并实现一个 LRU（最近最少使用）缓存淘汰算法，要求 get 和 put 操作的时间复杂度为 O(1)。请说明你的设计思路。",
			Difficulty: models.DifficultyHard, Type: models.QuestionTech},
		{CategoryID: techCats[3].ID, Title: "什么是时间复杂度和空间复杂度？如何分析一段代码的复杂度？",
			Content: "请解释算法分析中的时间复杂度和空间复杂度概念，常见的复杂度级别，以及如何分析一个算法的复杂度。",
			Difficulty: models.DifficultyEasy, Type: models.QuestionTech},

		// 系统设计
		{CategoryID: techCats[4].ID, Title: "如何设计一个短 URL 生成系统？",
			Content: "请设计一个类似 TinyURL 的短链接生成系统，需要涵盖系统架构设计、数据存储方案、哈希算法选择、重定向流程等。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionTech},
		{CategoryID: techCats[4].ID, Title: "如何设计一个支持海量用户的实时聊天系统？",
			Content: "请设计一个支持百万级用户的实时聊天系统，包括消息的实时推送、消息存储、离线消息处理、群聊功能等。",
			Difficulty: models.DifficultyHard, Type: models.QuestionTech},

		// 行为面试
		{CategoryID: nonTechCats[0].ID, Title: "请描述一次你解决过的技术难题",
			Content: "请用 STAR 法则描述一次你在工作中遇到的技术难题，你是如何分析问题、制定方案、最终解决问题的？",
			Difficulty: models.DifficultyMedium, Type: models.QuestionNonTech},
		{CategoryID: nonTechCats[0].ID, Title: "你是如何与团队成员协作完成一个复杂项目的？",
			Content: "请描述一个你参与过的需要多部门或多角色协作的复杂项目，你在这个项目中扮演了什么角色，遇到了哪些协作上的挑战？",
			Difficulty: models.DifficultyMedium, Type: models.QuestionNonTech},

		// HR 面试
		{CategoryID: nonTechCats[1].ID, Title: "你为什么想离开当前的公司？",
			Content: "当面试官问到你离开当前公司的原因时，你应该如何回答才能既诚实又不显得消极？请从职业发展的角度给出建议。",
			Difficulty: models.DifficultyEasy, Type: models.QuestionNonTech},
		{CategoryID: nonTechCats[1].ID, Title: "你期望的薪资是多少？如何回答薪资期望问题？",
			Content: "在面试中被问及期望薪资时，如何给出一个既能体现自身价值又不会让 HR 觉得不合理的回答？请给出具体的回答策略。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionNonTech},

		// 情景题
		{CategoryID: nonTechCats[2].ID, Title: "如果项目截止日期临近但核心功能还未完成，你会怎么做？",
			Content: "假设你负责的项目还有一个星期就要上线，但核心功能只完成了 60%，请描述你会如何应对这个情况。",
			Difficulty: models.DifficultyHard, Type: models.QuestionNonTech},
		{CategoryID: nonTechCats[2].ID, Title: "如果你的方案被上级否决了，你会怎么处理？",
			Content: "在工作中，你精心准备的方案被上级否决了，你会如何应对？请从情绪管理、沟通技巧和方案优化等角度进行分析。",
			Difficulty: models.DifficultyMedium, Type: models.QuestionNonTech},
	}

	for _, q := range questions {
		database.DB.Create(&q)
	}
	log.Printf("已创建 %d 道题目", len(questions))
}
