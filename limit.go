package antiDdos

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPRateLimiter представляет собой ограничитель скорости по IP-адресу.
type IPRateLimiter struct {
	ips map[string]*rate.Limiter // Карта IP-адресов и соответствующих ограничителей скорости
	mu  *sync.RWMutex            // Мьютекс для безопасного доступа к картам IP-адресов
	r   rate.Limit               // Предел скорости
	b   int                      // Максимальное количество запросов, разрешенных в интервал времени
}

// NewIPRateLimiter создает новый экземпляр IPRateLimiter с заданным интервалом времени и максимальным количеством запросов.
func NewIPRateLimiter(t time.Duration, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   rate.Every(t),
		b:   b,
	}

	return i
}

// AddIP добавляет IP-адрес в карту IP-адресов и создает для него новый ограничитель скорости.
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)

	i.ips[ip] = limiter

	return limiter
}

// GetLimiter возвращает ограничитель скорости для заданного IP-адреса.
// Если ограничитель для данного IP-адреса уже существует, он возвращается.
// В противном случае создается новый ограничитель и добавляется в карту IP-адресов.
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]

	if !exists {
		limiter = i.AddIP(ip)
	}

	return limiter
}

// LimitMiddleware является промежуточным обработчиком HTTP и ограничивает скорость запросов для каждого IP-адреса.
// Если количество запросов превышает максимальное количество разрешенных запросов в заданный интервал времени,
// возвращается ошибка "Too Many Requests" (код 429).
func (i *IPRateLimiter) LimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if ip == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		limiter := i.GetLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP возвращает IP-адрес клиента из заголовка X-Forwarded-For
// Если заголовок отсутствует, то IP-адрес извлекается из RemoteAddr.
func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")

	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	return ip
}
