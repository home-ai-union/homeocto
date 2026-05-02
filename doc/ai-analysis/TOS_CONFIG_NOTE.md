# TOS 上传功能配置说明

## 📌 当前状态

**TOS 上传功能已默认禁用** ✅

## 🔧 修改内容

### release.yml
```yaml
upload_tos:
  description: "Upload to Volcengine TOS"
  required: false
  type: boolean
  default: false  # 从 true 改为 false
```

## 💡 使用说明

### 当前行为
- 发布版本时，**默认不上传**到火山引擎 TOS
- 不需要配置 TOS 相关的 Secrets
- 节省配置时间，快速开始使用

### 何时启用 TOS 上传

当您有国内用户需要高速下载时，可以：

#### 方法 1: 手动触发时开启
1. 进入 Actions → Create Tag and Release
2. 点击 "Run workflow"
3. 将 **Upload to Volcengine TOS** 选项改为 `true`
4. 运行工作流

#### 方法 2: 修改默认值（永久启用）
编辑 `.github/workflows/release.yml`：
```yaml
upload_tos:
  default: true  # 改回 true
```

## 📋 启用前准备

如果决定启用 TOS 上传，需要：

1. **注册火山引擎账号**
   - 访问: https://www.volcengine.com/
   - 完成实名认证

2. **创建 TOS Bucket**
   - 名称: `homeocto-downloads`
   - 权限: 公共读
   - 区域: 华北2（北京）

3. **配置 GitHub Secrets**
   - `VOLC_TOS_ACCESS_KEY`: 访问密钥 ID
   - `VOLC_TOS_SECRET_KEY`: 访问密钥

4. **测试上传**
   ```bash
   aws s3 ls s3://homeocto-downloads/ \
     --endpoint-url https://tos-s3-cn-beijing.volces.com
   ```

## 🎯 建议

- **初期**: 保持禁用，专注功能开发
- **有国内用户后**: 启用 TOS 上传，提升下载速度
- **长期**: 根据用户分布决定是否继续使用

## 📊 TOS 费用参考

- 存储费用: ¥0.125/GB/月
- 下载流量: ¥0.5/GB
- 初期用户少时，月费用约 ¥5-20

---

**更新时间**: 2026-05-02  
**状态**: ✅ 已禁用（默认 false）
