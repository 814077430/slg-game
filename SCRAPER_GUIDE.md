# 高级爬虫工具包使用指南

## 功能特性

✅ **反反爬虫技术**：
- 随机用户代理轮换
- 人类行为模拟（随机延迟、滚动、鼠标移动）
- Stealth模式绕过检测
- 代理支持

✅ **多引擎支持**：
- **Requests + BeautifulSoup**：快速静态页面抓取
- **Selenium**：处理JavaScript渲染的动态页面
- **Puppeteer**：Node.js版本（高级功能）

✅ **智能重试机制**：
- 自动重试失败请求
- 反爬虫检测和应对
- 网络错误恢复

✅ **数据提取灵活**：
- CSS选择器支持
- 多种数据属性提取（文本、HTML、属性值）
- 批量数据处理

## 快速开始

### 1. 基础使用（静态页面）
```bash
python3 scraper-tool.py https://example.com --selectors "h1" "p" "a[href]"
```

### 2. 动态页面（JavaScript渲染）
```bash
python3 scraper-tool.py https://spa-example.com --selenium --selectors ".dynamic-content"
```

### 3. 自定义输出目录
```bash
python3 scraper-tool.py https://news-site.com --output-dir ./my-scrapes --selectors "h2.title" ".summary"
```

## 高级配置

### 反爬虫策略配置
编辑 `scraper-tool.py` 中的配置部分：

```python
# 反爬虫配置
ANTI_BOT_CONFIG = {
    'random_delays': True,           # 随机请求间隔
    'min_delay': 1,                  # 最小延迟（秒）
    'max_delay': 3,                  # 最大延迟（秒）
    'rotate_user_agents': True,      # 轮换用户代理
    'human_like_scrolling': True,    # 人类般的滚动行为
    'stealth_mode': True,            # 启用隐身模式
    'max_retries': 3,                # 最大重试次数
    'retry_delay': 5                 # 重试延迟（秒）
}
```

### 代理设置
如果需要使用代理，在代码中添加：

```python
PROXIES = {
    'http': 'http://your-proxy:port',
    'https': 'http://your-proxy:port'
}
```

## 使用示例

### 示例1：新闻网站标题抓取
```bash
python3 scraper-tool.py https://news.ycombinator.com --selectors ".storylink" ".score"
```

### 示例2：电商产品信息（动态加载）
```bash
python3 scraper-tool.py https://shop.example.com/products --selenium --selectors ".product-name" ".price" ".rating"
```

### 示例3：社交媒体数据
```bash
python3 scraper-tool.py https://twitter.com/username --selenium --selectors ".tweet-text" ".tweet-time"
```

## 输出格式

工具会生成以下文件：
- `scraped_data.json`：结构化数据
- `page_source.html`：原始HTML（可选）
- `screenshot.png`：页面截图（Selenium模式）

JSON格式示例：
```json
{
  "selector_0": ["标题1", "标题2", "标题3"],
  "selector_1": ["链接1", "链接2", "链接3"],
  "url": "https://example.com",
  "timestamp": "2024-02-03T16:30:00Z"
}
```

## 最佳实践

### 🛡️ 遵守robots.txt
```python
# 在代码中检查 robots.txt
import urllib.robotparser
rp = urllib.robotparser.RobotFileParser()
rp.set_url("https://example.com/robots.txt")
rp.read()
if not rp.can_fetch("*", url):
    print("被robots.txt禁止")
```

### ⏱️ 合理的请求频率
- 默认1-3秒延迟
- 高频请求可能导致IP被封
- 考虑使用代理池分散请求

### 🔍 错误处理
- 监控HTTP状态码
- 处理网络超时
- 记录失败URL以便重试

### 💾 数据存储
- 定期备份抓取的数据
- 使用数据库存储大量数据
- 考虑数据去重

## 故障排除

### 常见问题及解决方案

**Q: 被Cloudflare或其他WAF拦截**
- A: 启用 `--selenium` 模式
- A: 增加更长的随机延迟
- A: 使用高质量代理

**Q: JavaScript内容无法加载**
- A: 确保使用 `--selenium` 参数
- A: 增加页面等待时间

**Q: 选择器无法匹配元素**
- A: 使用浏览器开发者工具验证CSS选择器
- A: 尝试更通用的选择器

**Q: 内存占用过高**
- A: 减少并发请求数
- A: 及时关闭浏览器实例

## 扩展功能

### 自定义数据处理器
```python
def custom_processor(data, url):
    # 在这里添加自定义数据处理逻辑
    processed_data = {}
    for key, values in data.items():
        processed_data[key] = [clean_text(v) for v in values]
    return processed_data
```

### 批量URL处理
创建URL列表文件并循环处理：
```bash
while read url; do
    python3 scraper-tool.py "$url" --selectors "h1" "p"
    sleep 2
done < urls.txt
```

## 法律和道德考虑

⚠️ **重要提醒**：
- 遵守目标网站的使用条款
- 不要对服务器造成过大负载
- 尊重版权和隐私
- 某些网站明确禁止爬虫，请勿违反

## 性能优化

### 内存管理
- 及时关闭浏览器实例
- 使用无头模式减少资源占用
- 定期清理临时文件

### 网络优化
- 启用连接复用
- 设置合理的超时时间
- 使用缓存避免重复请求

---

这个爬虫工具包为你提供了从简单到复杂的完整爬虫解决方案。根据你的具体需求调整配置和选择器即可！

需要针对特定网站的爬虫策略吗？告诉我目标网站，我可以帮你定制解决方案。