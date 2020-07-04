как использовать?
Принимает время и сколько запросов в этот промежуток

```
limiter := antiDdos.NewIPRateLimiter(20*time.Second, 1)
```

Потом как хендлер можно использовать)

```
http.ListenAndServe(addr, limiter.LimitMiddleware(mux))
```
