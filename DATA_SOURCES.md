# 资料来源

本插件的固定资料由 `game-plugin-kit/scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留角色、装备名称、属性、技能文字与材料，不包含素材图片；伤害与评分由打包的上游脚本计算，见下文。

原神与星铁资料来自 miao-plugin，固定提交 7f6f1c84c89102bc6b1c58c8e1b06c61f4642161；保留 LICENSES/miao-plugin-MIT.txt。

没有账号时的面板来自 Mihomo（api.mihomo.me）的 `sr_info_parsed` 中文接口（miao-plugin 星铁的默认面板服务；上游读取未合计的 `sr_info` 后自行合计属性），按五分钟缓存；天赋等级同 miao 的 HomoApi。`templates/panel-list/` 与 `templates/artifact-list/` 按 miao-plugin 的 `character/profile-list` 与 `character/artis-list` 改写（MIT；miao 没有星铁角色的侧脸图，列表不显示头像）。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件及编译期业务库沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

## 装备评分

评分运行 miao 提交 `7f6f1c84c89102bc6b1c58c8e1b06c61f4642161` 的 `ArtisMarkCfg`、`ArtisMark` 与角色专属 `artis.js`，规则按当前属性、星魂、武器与套装自动选择，与上游一致。`internal/assets/calc/scores/<ID>-<角色名>.js` 由相邻库的 `scripts/bundle-reference-calculation.mjs` 从 `artis.js` 生成；默认权重（`artis-mark.js`）、主副词条表与套装简称随 `game.js` 打包。回归向量在 `internal/assets/testdata/score-vectors.json`。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存本游戏的角色脚本（`characters/<ID>-<角色名>.js`）、光锥与遗器效果（`game.js`）和计算资料（`catalog.json`），由相邻库的 `scripts/bundle-reference-calculation.mjs` 从 miao 固定快照生成；共用的 miao 运行时在相邻库 `reference/miao/`，打包修正见其中的 `PATCHES.md`。数值对照向量在 `internal/assets/testdata/calc-vectors.json`。

## 模拟与固定资料

材料、卡池与固定生日从同一 miao 固定快照的 material/info 数据转换，不联网补造日程。娱乐模拟的默认曲线来自 [Miao-Yunzai 固定提交](https://github.com/yoimiya-kokomi/Miao-Yunzai/tree/40cc2103efba1fbb279b768e3f5345d45372d357) 和 [StarRail-plugin 固定提交](https://github.com/TsukinaKasumi/StarRail-plugin/tree/090e411cf9be28b1644721fab54eab0ad283668f)，转换为 Go，不在运行时加载原插件。只向原神和星铁开放模拟入口；不声明为当前官方概率。

GPL-3.0 和 Apache-2.0 许可分别保留在 `LICENSES/Miao-Yunzai-GPL-3.0.txt` 与 `LICENSES/StarRail-plugin-Apache-2.0.txt`，对应的转换器、数据和原生实现随源码分发。转换模拟数据需要已有 Python/PyYAML，正常构建使用已生成数据，无需该再生成步骤。

云面板装备解释沿用同一固定 miao 快照的 artifact 元数据与 ArtisAttr 公式，包含原神 299、星铁 768 项装备映射。转换脚本为相邻业务库 `scripts/import-cloud-panel-data.mjs`，不加载上游机器人运行时；缺失档位明确显示未覆盖。

## 补充的强化公式

规则源文件在 `scripts/calc-supplements/`，打包后写入 `internal/assets/calc/characters/`。

固定快照只有强化花火（2306）和强化黑天鹅（2307）的资料，未提供计算脚本。`scripts/calc-supplements/` 根据同快照的技能、星魂和行迹描述补充规则，作者为本项目，不冒称原始脚本。强化花火按四层天赋、谜诡和战技目标为队友的情境计算；强化黑天鹅提供已减防、揭露与各奥迹层数情境。

强化黑天鹅终结技数值表的两行名称与参数位置颠倒：描述中的第一个参数对应直接伤害、第二个对应易伤。补充规则按描述位置解释，独立测试验证 120% 攻击倍率与 25% 易伤不会混用，并验证奥迹的 20% 无视防御。强化行迹优先关联资料中实际存在的 ID（包括额外的 `1` 前缀），不只依赖字符串替换。

## 图片模板

`templates/note/`、`templates/gacha/`、`templates/challenge/`、`templates/challenge-all/`、`templates/peak/`、`templates/peak-recent/`、`templates/help/`、`templates/month/`、`templates/rogue/`、`templates/swarm/`、`templates/divergent/` 与 `templates/divergent-run/` 按 StarRail-plugin 的 new_note、gatcha/new、challenge/index、index_all、index_peak、peak_recent、help/index（默认主题，图标按 config/system/help_system.js 中列出该命令的条目选取）、month、rogue/rogue、rogue/rogue_locust、rogue/rogueTourn 与 rogueTournNormal/rogueTournWeek 改写（常规演算与周期演算两页的样式只差周期信息一段，共用 rogueTournWeek 的样式；上游按 HTML 原样输出的面具说明与周期效果，这里只保留换行、颜色与加粗，其余标签转义或丢弃；黄金与机械、不可知域与货币战争上游没有出图，仍以文字回复）（月历的环形图上游用 ECharts 绘制，这里按相同的半径、圆心、圆角、配色与图例输出 SVG，不随插件分发该库）（Apache-2.0，见 `LICENSES/StarRail-plugin-Apache-2.0.txt`），跃迁统计同上游 GatchaData.stat，但卡池分组使用本插件随 miao 快照更新的卡池资料，而不是上游停在 2.4 版本的 poolData；持有率的分母读取下载的 StarRail-plugin baseData；`templates/ledger-count/` 按 Miao-Yunzai 原神插件的 html/ledger/ledger-count-sr 改写（StarRail-plugin 没有星琼统计页；GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），柱状图与环形图上游用 G2Plot 绘制，这里按相同的内边距、刻度算法、半径与标签规则输出 SVG；页头的星铁标志来自该插件仓库，素材来源为“原神插件图片”，只下载 `resources/StarRail/img/other/` 与 uid 页用到的 `resources/img/icon/`。`templates/calendar/` 与 `templates/calendar-list/` 按 miao-plugin 的 `wiki/calendar` 星铁模式改写（MIT；列表模式在上游缩窄页面，这里以第二份模板取其宽度），公告时间、跃迁图片、三种挑战的 42 天轮换与异相仲裁周期、排序合并的算法同上游 `apps/wiki/CalendarSr.js`；上游把轮换条目的类型写成函数，导致样式与图标缺失，这里按其所列类型出图。`templates/character-talent/` 按 miao-plugin 的 `wiki/character-talent` 星铁模式与 `common/tpl/talent-detail` 改写（MIT），技能、行迹与星魂资料运行时读取素材更新下载的 miao `meta-sr/character/<角色>/data.json`（为此 miao 素材来源也保留 `.json` 文件），描述中的数值按上游 getDesc 在 8 级（普攻与忆灵技能 5 级）代入，数字格式同 JavaScript 的 toFixed；描述 HTML 只保留标题、换行、斜体、加粗、nobr、span 与颜色。`templates/panel/` 的样式由 miao-plugin 固定提交的 `common/common.css` 与 `character/profile-detail.css` 转换而来，`templates/characters/` 与 `templates/training/` 按 miao 的 `character/avatar-list`、`character/profile-stat` 与通用角色卡片改写（MIT）。两者的图片地址都改为宿主渲染资源。模板用到的图片与字体不随插件分发，由管理员通过“素材更新”在运行时从上游仓库下载。深渊战绩里的角色、增益与首领图标是官方图片地址，首次出图时按需缓存到“官方图片缓存”。

`templates/uid-list/` 按 Miao-Yunzai 原神插件的 html/user/uid-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），页面框架与样式沿用 miao-plugin 的 common/layout/elem 与 common/common.css（MIT，见 `LICENSES/miao-plugin-MIT.txt`），同 miao 的 1.4 倍缩放；图片地址改为宿主渲染资源。玩家的头像与名片按 miao 的 faceImgs 取该 UID 已保存面板中编号最小的角色，没有时用 miao 的通用头像与名片。
