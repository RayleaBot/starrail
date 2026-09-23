# 资料来源

本插件的固定资料由 `scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留角色、装备名称、属性、技能文字与材料，不包含素材图片；伤害与评分由打包的上游脚本计算，见下文。

星铁资料来自 miao-plugin，固定提交 7f6f1c84c89102bc6b1c58c8e1b06c61f4642161；保留 LICENSES/miao-plugin-MIT.txt。

没有账号时的面板来自 Mihomo（api.mihomo.me）的 `sr_info_parsed` 中文接口（miao-plugin 星铁的默认面板服务；上游读取未合计的 `sr_info` 后自行合计属性），按五分钟缓存；天赋等级同 miao 的 HomoApi。`templates/panel-list/` 与 `templates/artifact-list/` 按 miao-plugin 的 `character/profile-list` 与 `character/artis-list` 改写（MIT；miao 没有星铁角色的侧脸图，列表不显示头像）。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

## StarRail-plugin 面板服务

“面板”“面板更新”等按 [StarRail-plugin 固定提交](https://github.com/TsukinaKasumi/StarRail-plugin/tree/090e411cf9be28b1644721fab54eab0ad283668f)的 `apps/panel.js` 查询其自带的第三方面板服务，服务列表取自同一提交的 `defSet/panelApi.yaml`；查询时向所选服务发送 UID 与签名，返回的面板属于该服务。签名由 `internal/panelapi/sign.b64` 计算，它是该提交 `resources/fuck/fuck` 的原样副本（由 `scripts/bundle-panel-api.mjs` 复制，未改写），与上游 `utils/auth.js` 的 getSign 一样解码 base64 后作为函数求值，只提供其模块引入的 md5。`templates/api-card/` 按同一提交的 `panel/new_card` 与 `common/common.css` 改写，同上游 1.6 倍缩放，角色头像取自“星铁插件图片”素材来源的 `resources/panel/resources/avatar/`。

角色面板的伤害表运行 `internal/panelapi/damage.js`：`scripts/bundle-panel-api.mjs` 把同一提交的 `utils/damage`（main.js 与 effect/ 下的角色、光锥、遗器效果）按原文转译为一个表达式，并内嵌其 data.js 读取的 SkillData、weapon_effect、Equipment、RelicSetSkill 与 character 表（character 表只保留 main.js 读取的各行迹第一级属性），logger.debug 不输出；技能的星魂提升读取原样复制的 `internal/panelapi/rank-skill-up.json`（`utils/damage/json/avatarRankSkillUp.json`）。`templates/api-panel/` 按 `panel/new_panel` 与 `common/common.css` 改写，同上游 1.6 倍缩放；页面脚本在 canvas 上画的行迹树改为同坐标的 SVG，背景 SVG 以 data 地址内嵌，图标按上游的 source-in 着色改用 SVG 滤镜。立绘、属性图标、星级、技能、遗器与光锥图片、行迹树背景和 `resources/panel/data/` 下的命途、遗器图标与行迹布局资料取自“星铁插件图片”素材来源（为此该来源也保留 `.svg` 文件；立绘共约 170 MB），行迹图标在出图时从 avocado.wiki 按需下载。修正上游的问题：服务给出的图标路径不以“/”开头时，上游拼出的地址缺少“/”而全部显示加载失败，这里补上；插件资料没有该角色（如记忆、欢愉命途与 1315 之后的角色）或命途没有行迹布局时，上游整页报错，这里照常出图，不画行迹树、不写命途；伤害模块的共享表会被计算改写（如瓦尔特六魂的战技段数每出一次图加一），这里每次重新载入；强化次数为 0 的副词条上游因 repeat(-1) 无法出图，这里不加标记。未移植其 PanelSetting 的面板原图开关、原图撤回与屏蔽面板图的群，原图沿用插件的“原图”规则。以上均为 Apache-2.0，见 `LICENSES/StarRail-plugin-Apache-2.0.txt`。

## 装备评分

评分运行 miao 提交 `7f6f1c84c89102bc6b1c58c8e1b06c61f4642161` 的 `ArtisMarkCfg`、`ArtisMark` 与角色专属 `artis.js`，规则按当前属性、星魂、光锥与套装自动选择，与上游一致。`internal/assets/calc/scores/<ID>-<角色名>.js` 由 `scripts/bundle-reference-calculation.mjs` 从 `artis.js` 生成；默认权重（`artis-mark.js`）、主副词条表与套装简称随 `game.js` 打包。回归向量在 `internal/assets/testdata/score-vectors.json`。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存本游戏的角色脚本（`characters/<ID>-<角色名>.js`）、光锥与遗器效果（`game.js`）和计算资料（`catalog.json`），由 `scripts/bundle-reference-calculation.mjs` 从 miao 固定快照生成；miao 运行时的适配在 `internal/reference/miao/`，打包修正见其中的 `PATCHES.md`。数值对照向量在 `internal/assets/testdata/calc-vectors.json`。

## 模拟与固定资料

材料与卡池从同一 miao 固定快照的 material/info 数据转换，不联网补造日程。娱乐模拟的默认曲线与物品池来自 [StarRail-plugin 固定提交](https://github.com/TsukinaKasumi/StarRail-plugin/tree/090e411cf9be28b1644721fab54eab0ad283668f)，转换为 Go，不在运行时加载原插件；不声明为当前官方概率。

Apache-2.0 许可保留在 `LICENSES/StarRail-plugin-Apache-2.0.txt`，对应的转换器、数据和原生实现随源码分发。

云面板装备解释沿用同一固定 miao 快照的 artifact 元数据与 ArtisAttr 公式，包含 768 项遗器映射。转换脚本为 `scripts/import-cloud-panel-data.mjs`，不加载上游机器人运行时；缺失档位明确显示未覆盖。

图鉴按 [Atlas 固定提交](https://github.com/Nwflower/atlas/tree/016e49357666e0823791abdf28fbb3b2efe68225) 的 `apps/atlas.js` 查找，各模块的规则取自其 `resource/Forlibrary/star-rail-atlas/rule_default/`，写在 `internal/assets/game.json`（GPL-3.0，见 `LICENSES/Atlas-GPL-3.0.txt`）。星铁图鉴库由管理员下载，不随插件分发；Atlas 自带的星铁别名文件以名称为键，而图鉴库以编号为键并自带全部模块的别名，因此不使用。

## 补充的强化公式

规则源文件在 `scripts/calc-supplements/`，打包后写入 `internal/assets/calc/characters/`。

固定快照只有强化花火（2306）和强化黑天鹅（2307）的资料，未提供计算脚本。`scripts/calc-supplements/` 根据同快照的技能、星魂和行迹描述补充规则，作者为本项目，不冒称原始脚本。强化花火按四层天赋、谜诡和战技目标为队友的情境计算；强化黑天鹅提供已减防、揭露与各奥迹层数情境。

强化黑天鹅终结技数值表的两行名称与参数位置颠倒：描述中的第一个参数对应直接伤害、第二个对应易伤。补充规则按描述位置解释，独立测试验证 120% 攻击倍率与 25% 易伤不会混用，并验证奥迹的 20% 无视防御。强化行迹优先关联资料中实际存在的 ID（包括额外的 `1` 前缀），不只依赖字符串替换。

## 图片模板

`templates/pool-info/` 按 miao-plugin 的 gacha/gacha-info 改写（MIT），卡池的筛选、每行八个与精简显示同上游 `apps/gacha/GachaPool.js`；`templates/gacha-detail/` 与 `templates/gacha-stat/` 按同一上游的 gacha/gacha-detail 与 gacha/gacha-stat 星铁模式改写（MIT），Miao-Yunzai 强制开启 miao 的 gachaStat 开关，StarRail-plugin 的跃迁分析规则之外不带“喵喵”的写法（如“角色池记录”“up统计”“全部统计”）同样出这两页，星铁没有的“集录”同上游读取角色跃迁与 UP 统计；出金、UP 判定与分期统计同上游 `apps/gacha/GachaData.js`，期次取自同一卡池资料，最后一期之后的“新版本”不设结束日期（上游写死的结束日期已过），落在各期之外的抽取之后更早的抽取仍按各自的期次统计（上游此后不再查找期次）；上游抽卡记录去掉的前缀不含“喵喵”，带“喵喵”的写法总是落到角色跃迁，这里一并去掉；页头头像上游按展柜的 headIcon 查找角色，星铁的头像编号不是角色编号，因此总是显示默认的卡芙卡，这里照做。`templates/character-card/` 按 miao-plugin 的 character/character-card 改写（MIT），横竖版的页面宽度与缩放同上游 `apps/character/AvatarCard.js`，触发写法同其 check；上游按角色的元素名给技能图标分类，没有与之对应的星铁样式，图标沿用页面默认的元素底色，这里照做。`templates/note/`、`templates/card/`、`templates/gacha/`、`templates/challenge/`、`templates/challenge-all/`、`templates/peak/`、`templates/peak-recent/`、`templates/help/`、`templates/month/`、`templates/rogue/`、`templates/swarm/`、`templates/divergent/`、`templates/divergent-run/`、`templates/currency-wars/` 与 `templates/currency-wars-archive/` 按 StarRail-plugin 的 new_note、card/card、gatcha/new、challenge/index、index_all、index_peak、peak_recent、help/index（默认主题，图标按 config/system/help_system.js 中列出该命令的条目选取）、month、rogue/rogue、rogue/rogue_locust、rogue/rogueTourn、rogueTournNormal/rogueTournWeek 与 gridFight/gridFight、gridFightArchive 改写（卡片同上游 apps/hkrpg.js 的 card，官方未设手机主题时换成上游指定的主题；上游的卡片规则没有结尾限定，会抢走“角色分析”“角色面板”等以卡片、探索、角色开头的写法，这里只接这三个词与紧随的 UID；“角色”在上游由这条规则先匹配，miao 的角色列表规则不接星铁前缀，因此同样出卡片；货币战绩按 apps/gridFight.js 取第几条记录，超出时同上游回复；上游在伤害记录找不到对应角色或羁绊时无法出图，这里只空出该项；常规演算与周期演算两页的样式只差周期信息一段，共用 rogueTournWeek 的样式；上游按 HTML 原样输出的面具说明与周期效果，这里只保留换行、颜色与加粗，其余标签转义或丢弃；黄金与机械与不可知域上游没有出图，仍以文字回复）（月历的环形图上游用 ECharts 绘制，这里按相同的半径、圆心、圆角、配色与图例输出 SVG，不随插件分发该库）（Apache-2.0，见 `LICENSES/StarRail-plugin-Apache-2.0.txt`），跃迁统计同上游 GatchaData.stat（上游选池时不去掉“统计”，规则收下的“角色统计”“光锥统计”等都落到总览，这里一并去掉），但卡池分组使用本插件随 miao 快照更新的卡池资料，而不是上游停在 2.4 版本的 poolData；持有率的分母读取“星铁插件图片”素材来源中的 StarRail-plugin baseData；`templates/ledger-count/` 按 Miao-Yunzai 原神插件的 html/ledger/ledger-count-sr 改写（StarRail-plugin 没有星琼统计页；GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），柱状图与环形图上游用 G2Plot 绘制，这里按相同的内边距、刻度算法、半径与标签规则输出 SVG；页头的星铁标志来自该插件仓库，素材来源为“原神插件图片”，只取 `resources/StarRail/img/other/`、星琼页用到的 `resources/StarRail/img/note/`、uid 页用到的 `resources/img/icon/`、抽卡记录与抽卡统计页用到的 `resources/img/other/bg4.png`、`resources/img/other/bg5.png` 与 `resources/font/HYWenHei-55W.ttf`，以及公告页用到的 `resources/html/mysNews/`、`resources/html/mysNews-list/`、`resources/font/tttgbnumber.ttf`。`templates/ledger/` 按同一上游的 html/ledger/ledger-sr 改写（GPL-3.0），月份的读取同 `model/ledger.js` 的 getMonth，数额与抽数同 dealData，环形图同样输出 SVG；genshinLayout 的横幅样式只作用于页头，这一页没有页头。getMonth 在一、二月把上一年的十一、十二月改成本月（判断后来月份的条件 `month <= 9 + month` 恒成立，星铁的 YYYYMM 月份又会记到今年），这里按上一年的月份查询。`templates/gacha-count/` 按同一上游的 html/gacha/log-count 星铁模式改写（GPL-3.0），分池、计数与排序同 `model/logCount.js`；上游固定提交的 `defSet/pool` 只到 2025 年，限定角色与光锥跃迁的期次改用本插件的卡池资料，常驻与新手沿用上游的单一卡池；Yunzai 的五星简称只有原神的，卡池名按全名显示；页头与上游一样显示 miao 给星铁角色的通用横幅。StarRail-plugin 的跃迁分析规则与 miao 的抽卡统计规则在上游先于它匹配，这一页只接其余写法，如“新手池统计”；上游规则里的“集录”在星铁读取不存在的 500 池而不出图，不收。`templates/gacha-log/` 与 `templates/gacha-all-log/` 按同一上游的 html/gacha/gacha-log 与 gacha-all-log 星铁模式改写（GPL-3.0），选池、统计、UP 判定与分色同 `model/gachaLog.js` 的 getPool、analyse、checkIsUp、randData 与 getAllGcLogData：单池页附加上游的 gacha-log-sr 样式，没有记录的卡池与上游一样显示全零；全部记录依次统计角色、角色联动、光锥、光锥联动与常驻跃迁（上游另读星铁没有的集录池），各块沿用第一块的标签颜色、五星时间段与 UP 标记，上游这一页漏了单池页附加的 gacha-log-sr 样式，标签因此没有底色，这里同单池页附加。修正上游的问题：最新一抽是四星时，上游把“未出四星”记为 0 后又被下一个四星覆盖，这里只在第一个四星处记录；上游边遍历边删除名为“未知”的五星，会跳过其后一个，这里全部删除后再计大保底；全部记录页上游把光锥联动按 90 抽着色，这里与其单池页一样按 80 抽。发送抽卡链接后的记录图同上游 logUrl，为角色跃迁的单池页。StarRail-plugin 的跃迁分析规则与 miao 的抽卡记录规则在上游先于它匹配，这两页只接其余写法，如“全部记录”“角色联动记录”“新手池记录”。`templates/calendar/` 与 `templates/calendar-list/` 按 miao-plugin 的 `wiki/calendar` 星铁模式改写（MIT；列表模式在上游缩窄页面，这里以第二份模板取其宽度），公告时间、跃迁图片、三种挑战的 42 天轮换与异相仲裁周期、排序合并的算法同上游 `apps/wiki/CalendarSr.js`；上游把轮换条目的类型写成函数，导致样式与图标缺失，这里按其所列类型出图。`templates/character-talent/` 按 miao-plugin 的 `wiki/character-talent` 星铁模式与 `common/tpl/talent-detail` 改写（MIT），技能、行迹与星魂资料读取 miao 素材来源的 `meta-sr/character/<角色>/data.json`（为此 miao 素材来源也保留 `.json` 文件），描述中的数值按上游 getDesc 在 8 级（普攻与忆灵技能 5 级）代入，数字格式同 JavaScript 的 toFixed；描述 HTML 只保留标题、换行、斜体、加粗、nobr、span 与颜色。`templates/panel/` 的样式由 miao-plugin 固定提交的 `common/common.css` 与 `character/profile-detail.css` 转换而来，`templates/training/` 按 miao 的 `character/profile-stat` 与通用角色卡片改写（MIT）。两者的图片地址都改为宿主渲染资源。模板用到的图片、字体与资料文件取自各上游的固定提交，由 `scripts/bundle-artwork.py` 复制到插件包的 `assets/` 随插件分发（许可同各上游，见 `LICENSES/`）；管理员可通过“素材更新”从上游仓库下载新版本，下载的文件优先使用。深渊战绩里的角色、增益与首领图标是官方图片地址，首次出图时按需缓存到“官方图片缓存”。

`templates/uid-list/` 按 Miao-Yunzai 原神插件的 html/user/uid-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），页面框架与样式沿用 miao-plugin 的 common/layout/elem 与 common/common.css（MIT，见 `LICENSES/miao-plugin-MIT.txt`），同 miao 的 1.4 倍缩放；图片地址改为宿主渲染资源。玩家的头像与名片按 miao 的 faceImgs 取该 UID 已保存面板中编号最小的角色，没有时用 miao 的通用头像与名片。

`templates/rank/` 按 miao-plugin 的 character/rank-profile-list 星铁模式改写（MIT），容器宽度同上游在页面内设置，群排行按 ark-plugin 改写后的宽度 930（欢愉命途 960、记忆命途 1000），并同其备份的 `miao-plugin-rank/resources/character/rank-profile-list.html` 加上全服排名一栏（再宽 180）；ark-plugin 的自定义排行沿用这一页面（MIT，见 `LICENSES/ark-plugin-MIT.txt`），同 `apps/customRank.js` 写入“全服数据”、排序与筛选说明及请求行数，页面宽 930（欢愉命途 960、记忆命途 1000），评级按六件遗器的平均分阈值（56 分以上为 MAX）。

`templates/panel/` 另按 ark-plugin 备份的 `miao-plugin-rank/resources/character/profile-detail.html`（MIT，见 `LICENSES/ark-plugin-MIT.txt`）加入面板排名：名次行接在伤害表后，排名统计块照搬其布局与内联样式，两张图按其 ECharts 选项（默认网格、smooth、面积渐变与标注）输出为 SVG，换装面板的伤害变化同其伤害表的写法（上游的内联样式改为同值的类）；渐变的色标按上游的原顺序输出（上游把红色色标写在前一色标之前，浏览器因此从本面板的位置起画 2.5 个百分点的红带，这里保持一致）；上游在星铁中同样写“圣遗物排名”“圣遗物评分”。

`templates/rank-stats/` 按 ark-plugin 的 graph/stats 改写（MIT，见 `LICENSES/ark-plugin-MIT.txt`），页面框架沿用 miao-plugin 的 common/layout/default 与 common/common.css，同 ark 的 1.4 倍缩放。上游在浏览器中用 ECharts 绘制折线，这里按 ECharts 的刻度与平滑算法输出相同的 SVG，不随插件分发该库；背景图取自“ark 插件图片”素材来源的 `resources/graph/`。

`templates/atlas-index/` 按 Atlas 的 resource/massage/text.html 与 text.css 改写（GPL-3.0，见 `LICENSES/Atlas-GPL-3.0.txt`），同上游的 1.5 倍缩放，数字字体取自 Yunzai 的 tttgbnumber；上游的 Source Han Sans CN Bold 字体超过宿主单个渲染资源 16 MiB 的上限，不随附，其余文字使用渲染器的字体。

`templates/charge/` 与 `templates/online/` 按 StarRail-plugin 的 charge 与 online 页面改写（Apache-2.0），“星琼记录”等与“在线时长”同上游 apps/hkrpg.js：资产记录读取官方自助查询的一页（每页 10 条）并只保留获取的记录（开拓力只保留多于 1 点的）；上游的“光锥记录”“武器记录”被跃迁记录的规则先匹配而从不出图，这里同样交给跃迁记录。在线时长读取开拓力记录中的“随时间回复开拓力”，按上游 statisticsOnlineDateGeneral 与 statisticOnlinePeriods 把每次单独回复的 1 点记为之前 6 分钟在线，逐日统计小时数并画出近七天的在线时段（上游逐分钟扫描使每段少记一分钟，这里照做；上游把零点后六分钟内的记录算到前一天时取错了日期，这里按其本意记到前一天）；每页 50 条，为在事件时限内回复最多读取 40 秒。两张 ECharts 图（近七天柱状图与日历热力图）按 ECharts 的默认布局、配色与分段输出 SVG，热力图上游固定为 2023 年，这里显示当前年份。

参考面板、收益曲线、强度榜、攻略、深渊攻略与商店光锥推荐同 StarRail-plugin 的 apps/srxx.js，直接发送该仓库 `resources/srsr/` 中的米游社作者整理图（Apache-2.0，见 `LICENSES/StarRail-plugin-Apache-2.0.txt`；上游在文件头注明各图作者），多图命令与上游一样以合并转发发送，平台不支持合并转发时改为一条消息。图片内容停在上游收录时的 1.x 版本，取自“星铁插件图片”素材来源；参考面板按角色的各个名字查找上游文件名，例如本插件的“穹·存护”对应上游的“开拓者•存护”。面板帮助同 miao-plugin 对“星铁面板帮助”的回复，发送该仓库的 `resources/character/imgs/help.jpg`（MIT），取自 miao 素材来源。

`templates/news/` 与 `templates/news-list/` 按 Yunzai 原神插件的 html/mysNews 与 html/mysNews-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），公告、资讯、活动、米游社搜索、帖子与预估按其 mysNews 的规则出图；详情页的样式去掉了米游社编辑器、加载、提示与回复控件等页面用不到的规则。米游社正文是任意 HTML，而渲染器可以联网，因此正文按页面样式用到的标签、类名与颜色字号等样式重建，图片只经渲染资源引用（官方图片缓存），链接与脚本不保留。与上游不同之处：正文中没有地址的超链图片按帖子的 structured_content 补上（上游显示为空白）；纯图片帖显示全部图片（上游只显示最后一张）；上游按 4000 像素分段截图，这里出一整张长图；二维码由 go-qrcode 生成（MIT，见 `LICENSES/go-qrcode-MIT.txt`），指向帖子所在游戏的米游社地址（上游固定为原神路径）。页面的米游社标志、图标字体、列表页背景与数字字体取自“原神插件图片”素材来源。星铁的“预估”同 StarRail-plugin 的 srEstimate，合并转发作者最新“星琼统计”帖的图片；“星琼预估”“盘点”等写法同 Yunzai 原神插件，按“星琼统计汇总”出图。
