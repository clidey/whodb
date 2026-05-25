# Semantic Test Contract Template

Create one semantic tag contract document for every UI module or workflow touched by semantic tag implementation. Prefer the filename `semantic-test-contract.md` unless the repository already has a different convention.

Before creating a new file, search for an existing module contract or testing documentation. Prefer extending the local convention over introducing a new docs shape.

Always create or update a contract when:

- Semantic tags are added, removed, renamed, or reviewed as part of the task.
- Existing tests, Playwright flows, black-box tests, evidence collectors, or GUI-agent workflows will depend on the tags.
- A requirement, PRD, or review request asks for automation-readable UI semantics.

If the repository has no obvious docs location, create a focused contract near the touched module or under the closest existing docs/test docs location. Keep it concise rather than skipping it.

If the repo only has a PR template and no module-level semantic contract convention, update or recommend the PR checklist as well, but still create a focused semantic tag contract for the touched module/workflow.

```markdown
# <Module> Semantic Test Contract

## 1. 模块信息

- 模块：
- 页面：
- 适用版本：
- 维护人：

## 2. 页面入口

| 页面 | 路由 | 说明 |
| --- | --- | --- |
|  |  |  |

## 3. 语义标签清单

| 页面元素 | 代码位置 | data-testid | 类型 | 业务语义 | data-qa-* | 可操作 | 可断言 | 证据来源 | 关联风险 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  | action / field / state / error / item / panel |  |  | 是 / 否 | 是 / 否 | requirement / route / API / state / permission / test |  |

## 4. 状态枚举

| 元素 | data-qa-state 可选值 | 说明 |
| --- | --- | --- |
|  |  |  |

## 5. 禁用原因枚举

| disabled reason | 含义 | 自动化预期 |
| --- | --- | --- |
|  |  |  |

## 6. 错误码枚举

| error code | 含义 | 自动化预期 |
| --- | --- | --- |
|  |  |  |

## 7. 资源绑定

| 页面元素 | 资源字段 | 说明 |
| --- | --- | --- |
|  |  |  |

## 8. 覆盖说明

| 需求元素 | 覆盖状态 | 说明 |
| --- | --- | --- |
|  | covered / skipped |  |

## 9. 变更规则

- 新增核心操作必须新增语义标签。
- 删除标签必须在 PR 中说明影响范围。
- 修改标签必须同步更新本文档。
- 自动化依赖的标签不得无通知重命名。
```

## Frontend PR Checklist

Add this checklist to frontend PR templates when the product has UI automation or GUI-agent testing:

```markdown
## UI 自动化语义标签检查

如果本 PR 涉及用户可操作页面，请确认：

- [ ] 核心操作按钮已添加稳定 `data-testid`
- [ ] 表单字段已添加稳定 `data-testid`
- [ ] 关键状态区域已添加可断言标签
- [ ] 错误提示 / 警告提示已添加可断言标签
- [ ] 禁用状态提供了明确原因，如 `data-qa-disabled-reason`
- [ ] 业务对象列表项提供了资源 ID，如 `data-qa-resource-id`
- [ ] 不依赖 CSS class、DOM 层级或按钮文案作为唯一自动化定位方式
- [ ] 新增 / 删除 / 重命名语义标签已同步更新 `semantic-test-contract.md`
```
