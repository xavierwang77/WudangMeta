package cmn

import (
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logDir = "logs"
)

var (
	logger     *zap.Logger
	MiniLogger *zap.Logger
	once       sync.Once = sync.Once{}
)

func InitLogger(debug bool) {
	once = sync.Once{}
	once.Do(func() {
		// 初始化日志文件目录
		err := initDir()
		if err != nil {
			logger.Fatal("init dir failed", zap.Error(err))
		}

		// 生成当前时间戳
		now := time.Now()
		timestamp := now.Format("2006-01-02T15-04-05")

		// 将时间戳插入到文件名中
		logFileName := fmt.Sprintf("%s/%s.log", logDir, timestamp)

		// 初始化日志
		if debug {
			err = initDevLogger()
			if err != nil {
				logger.Fatal("init dev logger failed" + err.Error())
			}
		} else {
			err = initProdLogger(logFileName)
			if err != nil {
				logger.Fatal("init prod logger failed" + err.Error())
			}
		}

		// 初始化极简日志
		err = initMiniLogger()
		if err != nil {
			logger.Fatal("init mini logger failed" + err.Error())
		}

		logger = zap.L()
	})

	MiniLogger.Info("[ OK ] log module initialized")
}

// GetLogger 获取全局的logger
func GetLogger() *zap.Logger {
	return logger
}

// initDevLogger 初始化开发环境日志
func initDevLogger() error {
	// 使用带颜色的控制台编码器
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "T"   // 时间字段仍然叫 T
	encoderConfig.CallerKey = "C" // 调用者字段仍然叫 C

	// 关键在这里，把级别编码改为带颜色版本
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// 你也可以根据需要设置 TimeEncoder、CallerEncoder，例如：
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeCaller = zapcore.FullCallerEncoder

	// 创建一个 ConsoleEncoder，用它输出到控制台，这样就会根据 EncodeLevel 自动给日志级别加颜色
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// 控制台输出级别写 Debug，文件输出级别写 Error
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

	core := zapcore.NewTee(consoleCore)
	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)

	return nil
}

// initProdLogger 初始化生产环境日志
func initProdLogger(logFilePath string) error {
	// 参数校验
	if logFilePath == "" {
		fmt.Println("log file path is empty, init log failed")
		return nil
	}

	// 创建日志文件（如果文件已经存在，会覆盖；在实际项目中常配合日志切割工具一起使用）
	file, err := os.Create(logFilePath)
	if err != nil {
		fmt.Printf("create log file failed: %v\n", err)
		return err
	}

	// 使用生产环境的 EncoderConfig
	encoderConfig := zap.NewProductionEncoderConfig()
	// 生产环境中一般使用 ISO8601 时间格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 生产环境通常记录短路径调用者
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	// 生产环境通常使用字符串格式的 Duration 编码
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	// 生产环境不使用彩色级别编码，保留默认
	// encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 对控制台输出，仍然使用 JSONEncoder，但只打印 Warn 及以上级别
	consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel, // 仅在控制台打印 WARN 及以上
	)

	// 对文件输出，使用 JSONEncoder，记录 Info 及以上
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		zapcore.InfoLevel, // 写入文件的最小级别
	)

	// 将控制台和文件两个 Core 合并
	core := zapcore.NewTee(consoleCore, fileCore)

	// 生产环境一般去掉开发模式下的 Stacktrace 太多信息，直接启用 caller 和时间戳即可
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
	zap.ReplaceGlobals(logger)

	return nil
}

func initMiniLogger() error {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg", // 保留 msg
		EncodeTime:   nil,   // 不显示时间
		EncodeLevel:  nil,   // 不显示 level
		EncodeCaller: nil,   // 不显示 caller
	}

	// 使用 console 输出（非 JSON）
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(zapcore.Lock(os.Stdout)),
		zapcore.InfoLevel, // 只输出 info 及以上
	)

	MiniLogger = zap.New(core)

	return nil
}

func initDir() error {
	// 创建日志目录
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		fmt.Printf("create logs directory failed: %v\n", err)
		return err
	}

	return nil
}
