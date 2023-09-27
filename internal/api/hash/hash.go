package hash

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/greyfox12/Gophermart/internal/api/logmy"
)

// MD5 hash
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// генерируем случайную последовательность байт
func generateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type AuthGen struct {
	Secretkey []byte
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthGen) Init() error {
	var err error
	if h.Secretkey, err = generateRandom(32); err != nil {
		logmy.OutLog(fmt.Errorf("error generateRandom:  %w", err))
		return err
	}
	return nil
}

// Генерирую токен пользователя
type Claims struct {
	jwt.RegisteredClaims
	UserLogin string
}

func (h *AuthGen) CreateToken(login string) (string, error) {

	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		// собственное утверждение
		UserLogin: login,
	})

	tokenString, err := token.SignedString(h.Secretkey)
	if err != nil {
		return "", err
	}

	//	fmt.Printf("tokenString=%v\n", tokenString)
	// возвращаем строку токена
	return tokenString, nil
}

func (h *AuthGen) GetUserId(tokenString string) string {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return h.Secretkey, nil
		})
	if err != nil {
		logmy.OutLog(err)
		return ""
	}

	if !token.Valid {
		fmt.Println("Token is not valid")
		return ""
	}

	fmt.Println("Token os valid")
	return claims.UserLogin
}

func ValidLuna(number int64) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int64) int64 {
	var luhn int64

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}

func ValidLunaStr(vpan string) bool {

	x := 0
	s := 0
	for i, r := range strings.Split(vpan, "") {
		x, _ = strconv.Atoi(r)
		if i%2 != 0 {
			x = x * 2
			if x > 9 {
				x = x - 9
			}
		}
		s = s + x
	}
	s = 10 - s%10
	if s == 10 {
		s = 0
	}
	return s == 0
}

// Проверяю токен
func (h *AuthGen) CheckAuth(token string) (string, int) {
	if token == "" {
		logmy.OutLog(fmt.Errorf("checkauth: no autorization head"))
		return "", 401
	}

	token_buf := strings.Split(token, " ")
	if len(token_buf) != 2 {
		logmy.OutLog(fmt.Errorf("checkauth: unknow format autorization head: %w", token))
		return "", 401
	}

	if token_buf[0] != "Bearer" {
		logmy.OutLog(fmt.Errorf("orders: unknow type autorization head: %w", token_buf[0]))
		return "", 401
	}

	login := h.GetUserId(token_buf[1])
	if login == "" {
		logmy.OutLog(fmt.Errorf("orders: unknow type autorization head: %w", token_buf[0]))
		return "", 401
	}

	return login, 0
}
