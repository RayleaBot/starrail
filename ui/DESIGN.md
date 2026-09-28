---
name: 星穹列车
description: 深空群青画布、暖象牙文字、金色主操作与切角几何组成的星铁插件管理页
colors:
  space-950: "#0b0f1a"
  space-900: "#111729"
  space-800: "#161d31"
  space-700: "#1c2540"
  space-600: "#25304f"
  line: "#2a3350"
  line-strong: "#3d4a70"
  ivory: "#f1ede3"
  ivory-muted: "#a8b0c3"
  ivory-faint: "#7f889f"
  gold-300: "#f2d48c"
  gold-400: "#e5c479"
  gold-500: "#c9a95e"
  gold-700: "#7d6a3c"
  on-gold: "#1a1406"
  gold-soft: "#2a2740"
  sky-400: "#7cb2e6"
  sky-soft: "#1b2a45"
  success: "#7fd6a4"
  success-soft: "#16302a"
  warning: "#f0bb5a"
  warning-soft: "#332a1a"
  danger: "#ff8a8a"
  danger-soft: "#3a2226"
  attention: "#f4b56d"
  attention-soft: "#33281d"
typography:
  headline:
    fontFamily: "'Barlow Semi Condensed', 'Microsoft YaHei UI', 'PingFang SC', system-ui, sans-serif"
    fontSize: "22px"
  title:
    fontFamily: "'Barlow Semi Condensed', 'Microsoft YaHei UI', 'PingFang SC', system-ui, sans-serif"
    fontSize: "17px"
  section:
    fontFamily: "'Barlow Semi Condensed', 'Microsoft YaHei UI', 'PingFang SC', system-ui, sans-serif"
    fontSize: "15px"
  body:
    fontFamily: "'Barlow Semi Condensed', 'Microsoft YaHei UI', 'PingFang SC', 'Hiragino Sans GB', system-ui, sans-serif"
    fontSize: "14px"
  label:
    fontFamily: "'Barlow Semi Condensed', 'Microsoft YaHei UI', 'PingFang SC', 'Hiragino Sans GB', system-ui, sans-serif"
    fontSize: "13px"
  caption:
    fontFamily: "'Barlow Semi Condensed', 'Microsoft YaHei UI', 'PingFang SC', 'Hiragino Sans GB', system-ui, sans-serif"
    fontSize: "12px"
  metric:
    fontFamily: "'Barlow Semi Condensed', 'Microsoft YaHei UI', 'PingFang SC', system-ui, sans-serif"
    fontSize: "20px"
  mono:
    fontFamily: "'Barlow Semi Condensed', 'Cascadia Mono', Consolas, monospace"
    fontSize: "13px"
rounded:
  xs: "2px"
  sm: "3px"
  md: "4px"
  lg: "6px"
  full: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "24px"
  xxl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.gold-400}"
    textColor: "{colors.on-gold}"
    rounded: "{rounded.sm}"
    height: "36px"
  button-secondary:
    backgroundColor: "{colors.space-700}"
    textColor: "{colors.ivory}"
    rounded: "{rounded.sm}"
    height: "36px"
  input:
    backgroundColor: "{colors.space-900}"
    textColor: "{colors.ivory}"
    rounded: "{rounded.sm}"
    height: "36px"
  navigation-item:
    backgroundColor: "{colors.gold-soft}"
    textColor: "{colors.gold-300}"
    rounded: "{rounded.sm}"
  status-chip:
    backgroundColor: "{colors.success-soft}"
    textColor: "{colors.success}"
    rounded: "{rounded.xs}"
  section-surface:
    backgroundColor: "{colors.space-800}"
    textColor: "{colors.ivory}"
    rounded: "{rounded.lg}"
  data-row:
    backgroundColor: "{colors.space-800}"
    textColor: "{colors.ivory}"
---

# Design System: 星穹列车

## Overview

**Creative North Star: "星穹列车的信息面板"**

页面借用《崩坏：星穹铁道》官方界面的语言：深空群青的画布带极淡的星点与顶部辉光，信息面板是略浅一级的群青，边界是 1px 细线而不是阴影，重要边角切成 45° 斜角，金色只落在主操作、当前位置和关键数值上，天青蓝负责链接与焦点。文字是暖象牙白，密度接近游戏内的角色与档案面板。

它服务于 Operate 场景：操作者选好角色后立即看到结果，页面不做营销式的大标题与巨型数字。

**Key Characteristics:**

- 单一深色主题，不随宿主亮暗切换；`forced-colors` 下退回系统颜色。
- 金色的主按钮、当前分区与数值高亮；其余控件为群青面上的象牙文字与细线。
- 切角几何：实底按钮（主操作与破坏性操作）的左上、右下角切 8px，小尺寸切 4px；结果面板左上角切 8px 且切边镀金；当前导航项右上角切 4px；输入框保持小圆角。
- 全页使用自带的 Barlow Semi Condensed（OFL，随包分发）处理拉丁字母与数字，中文由同一字体栈回退到系统无衬线，同一面板内不会出现两种数字。
- 选择即查询：没有额外输入的读取在选择后自动执行。

## Colors

- **画布与面板**：space-950 为画布，space-800 为面板与表单区域，space-700 为浮起表面与次级按钮，space-600 为悬停面。画布叠加两层极淡的径向辉光（金与天青，各不超过 8% 不透明度）和一层稀疏星点，星点由 CSS 径向渐变绘制，不引入图片。
- **文字**：ivory 正文，ivory-muted 说明与元数据，ivory-faint 占位与禁用。
- **金**：gold-400 主按钮、当前分区、面板切角、关键数值；gold-300 悬停；gold-500 按下；gold-700 用于细线强调。金不用于大面积背景，也不用于面板标题或非数值的指标文字。
- **天青**：sky-400 用于链接、字段焦点与选中项的辅助标记。
- **语义**：success、warning、danger、attention 各有独立的文字色和 soft 背景，状态同时给出文字或图标。

**The Gold Budget Rule.** 一屏之内金色只出现在主操作、当前位置、关键数值和分隔线强调上；装饰、普通边框和普通文字不用金。

**The Line Over Shadow Rule.** 面板与控件靠 1px 细线和明度差分层；阴影只用于真正覆盖内容的浮层（对话框）。

## Typography

- 全页字体栈以 Barlow Semi Condensed 打头（`unicode-range` 只覆盖拉丁区），中文回退到系统无衬线；标题 600 字重，字号 22 / 17 / 15。
- 正文 14px，标签与说明 13px，芯片、页脚与表格内注释 12px；指标数值 20px。数字一律 `font-variant-numeric: tabular-nums`。
- UID、任务 ID、命令用法等标识符用 Barlow Semi Condensed 500，不用等宽字体做“技术感”装饰。

## Layout

- 内容区最大宽 1440px，在宿主 iframe 内居中；内部所有文字与控件左对齐。
- 工作区：标题行（标题 + 一句说明，右侧共用的角色选择器）；下方左侧 224px 的分区导航，右侧内容列一次只显示一个分区，分区内表单在前、结果在后。
- 分区导航项左对齐，当前项左侧有 2px 金色竖线、右上角切 4px、文字金色、背景 gold-soft。
- ≤ 760px：导航改为横向滑动条，两端渐隐；≤ 720px 或粗指针：控件 44px 高。

## Shapes

- 实底按钮（主操作、破坏性操作）：`clip-path` 切左上与右下角 8px；紧凑尺寸切 4px。
- 结果面板：左上角切 8px，其余三角 `rounded.lg`；外层画 1px 细线、内层为内缩 1px 的同形填充，切边不丢描边；切边与相邻约 10px 的边线镀金，作为面板的角标。只用于独立结果面板与对象，不用于表单区。
- 空态框：`rounded.lg` 圆角与 1px 细线。输入框、下拉、芯片：`rounded.sm` / `rounded.xs`。
- 工作区标题下的 56px 金线为 2px 高、末端 45° 收尾。

## Components

- **按钮**：主操作金色实底 + on-gold 文字；次级为 space-700 面 + 细线；危险为 danger 文字与 danger-soft 悬停；破坏性实底为 danger 底 + 深色文字。禁用 45% 不透明度。焦点为内缩 2px sky-400 轮廓（金色按钮上为 on-gold）。
- **字段**：标签 13px 象牙；输入框 space-900 底、line 边框、聚焦时 sky-400 边框；错误时 danger 边框并附文字。
- **导航项**：见 Layout。
- **状态行**：图标 + 文字，语义色只染图标与关键文字。
- **芯片**：soft 背景 + 语义文字 + 6px 圆点。
- **数据行**：细线分隔，主文字象牙 500 字重，元数据 ivory-muted 13px，操作按钮靠右。
- **表格**：表头 space-700 底、ivory-muted 文字、粘性；行悬停 space-700。
- **面板**：space-800 底、line 描边、左上切角镀金，用于查询结果与独立对象；面板标题为象牙色，金只在切角上。
- **空态**：细线框带极淡的金色顶部渐变，金色图标 + 一句话 + 下一步说明。

## Motion

- 分区切换 200ms 淡入（`cubic-bezier(0.16, 1, 0.3, 1)`），按钮背景 120ms；`prefers-reduced-motion` 下全部关闭。
- 不做入场编排、不做悬停位移。

## Do's and Don'ts

- Do：让没有额外输入的查询在选择后自动执行；把结果放在表单下方；用切角与细线分层。
- Don't：圆角卡片墙、嵌套卡片、发光边框、渐变文字、大面积金色、把主表单藏进折叠块、依赖宿主主题变量。
