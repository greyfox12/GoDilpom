package logmy

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stdout", "./logs.txt"}
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder

	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}

// RequestLogger — middleware-логер для входящих HTTP-запросов.
func RequestLogger(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
			responseData:   responseData,
		}

		h(&lw, r)

		duration := time.Since(start)

		Log.Info("got incoming HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Duration("duration", duration),
			zap.Int("size", responseData.size),
			zap.Int("status", responseData.status),
		)
		Log.Sync()
		//	fmt.Printf("Write log\n")

	})
}

func OutLogDebug(errorStr error) {
	Log.Debug("Debug:",
		zap.String("Message", fmt.Sprint(errorStr)),
	)
	Log.Sync()
}

func OutLogInfo(errorStr error) {
	Log.Info("Info:",
		zap.String("Message", fmt.Sprint(errorStr)),
	)
	Log.Sync()
}

func OutLogWarn(errorStr error) {
	Log.Warn("Warn:",
		zap.String("Message", fmt.Sprint(errorStr)),
	)
	Log.Sync()
}

func OutLogError(errorStr error) {
	Log.Error("Error:",
		zap.String("Message", fmt.Sprint(errorStr)),
	)
	Log.Sync()
}

func OutLogFatal(errorStr error) {
	Log.Fatal("Fatal:",
		zap.String("Message", fmt.Sprint(errorStr)),
	)
	Log.Sync()
}

/*func OutLog(errorStr error) {
	Log.Fatal("Fatal:",
		zap.String("Message", fmt.Sprint(errorStr)),
	)
	Log.Sync()
}
*/
