package generator

import "github.com/tlipoca9/devgen/genkit"

// config returns the tool configuration.
func (g *Generator) config() genkit.ToolConfig {
	return genkit.ToolConfig{
		OutputSuffix: "_delegator.go",
		Annotations: []genkit.AnnotationConfig{
			{
				Name: "delegator",
				Type: "type",
				Doc: `为接口生成委托器（装饰器模式）。

用法：
  // delegatorgen:@delegator
  type UserRepository interface {
      GetByID(ctx context.Context, id string) (*User, error)
      Save(ctx context.Context, user *User) error
  }

生成内容：
  - Builder 模式的委托器构建器
  - 可选的 Delegator：Tracing、Cache
  - 所有 Delegator 接口内联定义，零外部依赖（Tracing 除外，使用 OTel）

使用示例：
  repo := NewUserRepositoryDelegator(baseRepo).
      WithTracing(tracer).
      WithCache(cache).
      Build()`,
			},
			{
				Name: "trace",
				Type: "field",
				Doc: `配置方法的链路追踪行为。

用法：
  // delegatorgen:@trace                    // 启用，默认 span 名
  // delegatorgen:@trace(span=CustomName)   // 自定义 span 名
  // delegatorgen:@trace(attrs=id,name)     // 记录参数为属性
  // 无注解 = 跳过此方法（直接透传）`,
				Params: &genkit.AnnotationParams{
					Docs: map[string]string{
						"span":  "自定义 span 名称（默认：接口名.方法名）",
						"attrs": "记录为属性的参数名（逗号分隔）",
					},
				},
			},
			{
				Name: "cache",
				Type: "field",
				Doc: `配置方法的缓存行为（仅适用于有返回值的方法）。

用法：
  // delegatorgen:@cache                                     // 启用（默认配置）
  // delegatorgen:@cache(ttl=10m)                            // 自定义 TTL
  // delegatorgen:@cache(key="user:{id}")                    // 自定义 key 后缀
  // delegatorgen:@cache(prefix="myapp:")                    // 自定义 key 前缀
  // delegatorgen:@cache(ttl=5m, jitter=15)                  // 自定义 jitter（±15%）
  // delegatorgen:@cache(ttl=5m, refresh=30)                 // TTL 剩余 30% 时异步刷新
  // delegatorgen:@cache(ttl=5m, refresh=0)                  // 禁用异步刷新

Key 生成：
  完整 key = prefix + key
  默认 prefix：{PKG}:{INTERFACE}:
  默认 key：{METHOD}:{base64_json()}
  例如：github.com/myapp/user:UserRepository:GetByID:eyJpZCI6IjEyMyJ9

内置变量：
  {PKG}       - 完整包 import 路径
  {INTERFACE} - 接口名
  {METHOD}    - 方法名

参数引用：
  {param}       - 引用方法参数
  {param.Field} - 引用参数的字段

内置函数：
  {base64_json()}          - 所有非 context 参数 JSON+Base64
  {base64_json(p1, p2)}    - 指定参数 JSON+Base64

高级特性（自动检测启用）：
  - Jitter: TTL 抖动，防止缓存雪崩（默认 ±10%）
  - 异步刷新: Cache 实现 CacheAsyncExecutor 接口时自动启用
  - 错误缓存: Cache.SetError() 方法决定是否缓存错误，防止缓存穿透
  - 分布式锁: Cache 实现 CacheLocker 接口时自动启用，防止缓存击穿`,
				Params: &genkit.AnnotationParams{
					Docs: map[string]string{
						"prefix":  "缓存 key 前缀模板（默认 {PKG}:{INTERFACE}:）",
						"key":     "缓存 key 后缀模板（默认 {METHOD}:{base64_json()}）",
						"ttl":     "缓存过期时间（如 5m, 1h，默认 5m）",
						"jitter":  "TTL 抖动百分比（默认 10，即 ±10%）",
						"refresh": "异步刷新阈值百分比（默认 20，设为 0 禁用）",
					},
				},
			},
			{
				Name: "cache_evict",
				Type: "field",
				Doc: `配置方法执行后驱逐的缓存 key。

用法：
  // delegatorgen:@cache_evict(key="user:{id}")
  // delegatorgen:@cache_evict(keys="user:{id},user:list:{id}")

注意：key 和 keys 参数互斥，只能使用其中一个。`,
				Params: &genkit.AnnotationParams{
					Docs: map[string]string{
						"key":  "要驱逐的单个缓存 key 模板",
						"keys": "要驱逐的多个缓存 key 模板（逗号分隔）",
					},
				},
			},
		},
	}
}
