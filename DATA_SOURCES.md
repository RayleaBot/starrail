# 资料来源

本插件的固定资料由 `scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留角色与光锥的名称、命途、80 级基础属性、技能文字（角色技能按普攻 6 级、其余 10 级填入数值，光锥技能写叠影 1 到 5 的区间）、星魂文字与材料，不包含素材图片；伤害与评分由打包的上游脚本计算，见下文。

星铁图鉴、别名、材料、卡池、云面板装备资料、伤害与评分来自 miao-plugin，固定提交 c9fe416ea8d22140101a33246dc380a140e7e068；保留 LICENSES/miao-plugin-MIT.txt。

固定快照目录为 `<参考项目/2026-10-13>`。StarRail-plugin 固定在 `3e68a03fccde6e00100536b102e93b31d609a1cf`，ark-plugin 固定在 `bdced6bf8b1821e0b5e5a96ed4d7e78beda089a1`；Miao-Yunzai 保持 `40cc2103efba1fbb279b768e3f5345d45372d357`，Yunzai-genshin 保持 `4a2e1fb8f094b2a8f039ce0768773701718b60b9`，Atlas 保持 `016e49357666e0823791abdf28fbb3b2efe68225`，对应的 `*.source.json` 随快照保存。

没有账号时的面板来自 Mihomo（api.mihomo.me）的 `sr_info_parsed` 中文接口（miao-plugin 星铁的默认面板服务；上游读取未合计的 `sr_info` 后自行合计属性），按五分钟缓存；天赋等级同 miao 的 HomoApi。`templates/panel-list/` 与 `templates/artifact-list/` 按 miao-plugin 的 `character/profile-list` 与 `character/artis-list` 改写（MIT；miao 没有星铁角色的侧脸图，列表不显示头像）。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

管理页的拉丁字母与数字使用 [Barlow Semi Condensed](https://github.com/jpt/barlow)（SIL Open Font License 1.1，见 `LICENSES/Barlow-OFL-1.1.txt`），只随包分发 500 与 600 两个字重的拉丁子集，位于 `ui/src/assets/fonts/`；中文使用系统字体。管理页的配色与几何参照《崩坏：星穹铁道》官方界面语言，不分发游戏的字体、图片或商标，见 `ui/DESIGN.md`。

## 装备评分

评分运行 miao 提交 `c9fe416ea8d22140101a33246dc380a140e7e068` 的 `ArtisMarkCfg`、`ArtisMark` 与角色专属 `artis.js`，规则按当前属性、星魂、光锥与套装自动选择，与上游一致。`internal/assets/calc/scores/<ID>-<角色名>.js` 由 `scripts/bundle-reference-calculation.mjs` 从 `artis.js` 生成；默认权重（`artis-mark.js`）、主副词条表与套装简称随 `game.js` 打包。评分按实际主词条计算部位上限，单件封顶 70，同上游。回归向量在 `internal/assets/testdata/score-vectors.json`。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存本游戏的角色脚本（`characters/<ID>-<角色名>.js`）、光锥与遗器效果（`game.js`）和计算资料（`catalog.json`），由 `scripts/bundle-reference-calculation.mjs` 从 miao 固定快照生成；miao 运行时的适配在 `internal/reference/miao/`，打包修正见其中的 `PATCHES.md`。真珠的治疗字段与 `calc(attr.def)` 直接使用上游实现；刻律德菈的表行名称保留本地修正。数值对照向量在 `internal/assets/testdata/calc-vectors.json`。

## 模拟与固定资料

材料与卡池从同一 miao 固定快照的 material/info 数据转换，不联网补造日程。材料使用新的 boss/talent/normal/weekly/virtual 分类，同系列取最低星材料名称；常规卡池收录至 4.6 下半，卡池名直接取 miao 的 `poolNameSr`。卡池按版本倒序，同版本按开始时间倒序；这与上游仅将“下半”排在前面、其他半期保持原序的排序不同，3.8 因而按下半、中场、上半排列。跃迁记录页的两期 Fate 联动池来自同一 StarRail-plugin 固定提交的 `tools/collab-pools.json`，独立存放在 `collab_pools`。娱乐模拟的默认曲线、物品池与联动池来自 [StarRail-plugin 固定提交](https://github.com/TsukinaKasumi/StarRail-plugin/tree/3e68a03fccde6e00100536b102e93b31d609a1cf)，转换为 Go，不在运行时加载原插件；不声明为当前官方概率。

Apache-2.0 许可保留在 `LICENSES/StarRail-plugin-Apache-2.0.txt`，对应的转换器、数据和原生实现随源码分发。

云面板装备解释沿用同一固定 miao 快照的 artifact 元数据与 ArtisAttr 公式，包含 768 项遗器映射。转换脚本为 `scripts/import-cloud-panel-data.mjs`，不加载上游机器人运行时；缺失档位明确显示未覆盖。

图鉴按 [Atlas 固定提交](https://github.com/Nwflower/atlas/tree/016e49357666e0823791abdf28fbb3b2efe68225) 的 `apps/atlas.js` 查找，各模块的规则取自其 `resource/Forlibrary/star-rail-atlas/rule_default/`，写在 `internal/assets/game.json`（GPL-3.0，见 `LICENSES/Atlas-GPL-3.0.txt`）。星铁图鉴库由管理员下载，不随插件分发；固定快照的图鉴库收录至 4.4；Atlas 自带的星铁别名文件以名称为键，而图鉴库以编号为键并自带全部模块的别名，因此不使用。

## 补充的强化公式

规则源文件在 `scripts/calc-supplements/`，打包后写入 `internal/assets/calc/characters/`。

固定快照只有强化花火（2306）和强化黑天鹅（2307）的资料，未提供计算脚本。`scripts/calc-supplements/` 根据同快照的技能、星魂和行迹描述补充规则，作者为本项目，不冒称原始脚本。强化花火按四层天赋、谜诡和战技目标为队友的情境计算；强化黑天鹅提供已减防、揭露与各奥迹层数情境。

强化黑天鹅终结技数值表的两行名称与参数位置颠倒：描述中的第一个参数对应直接伤害、第二个对应易伤。补充规则按描述位置解释，独立测试验证 120% 攻击倍率与 25% 易伤不会混用，并验证奥迹的 20% 无视防御。强化行迹优先关联资料中实际存在的 ID（包括额外的 `1` 前缀），不只依赖字符串替换。

## 图片模板

`templates/gacha-simulation/` 按 StarRail-plugin 的 gachasimulation 改写（Apache-2.0），立绘、光锥图、命途与属性图标来自 `resources/gachasimulation/resources/images/`，记忆、欢愉使用上游 weaponlife 图；缺少的角色或光锥使用 miao 立绘或图标。

`templates/pool-info/` 按 miao-plugin 的 gacha/gacha-info 改写（MIT），卡池的筛选、每行八个与精简显示同上游 `apps/gacha/GachaPool.js`，卡池按版本倒序、同版本按开始时间倒序；按名称查询时页首提示正在 UP 或已多少天未复刻，尚未开始的期次不计入未复刻天数；`templates/gacha-stat/` 按同一上游的 gacha/gacha-stat 星铁模式改写（MIT），是抽卡统计的页面，回答以“统计”结尾的跃迁写法：不写卡池时同上游统计角色与光锥活动跃迁，写角色或光锥时只统计该跃迁，均按期次分开；常驻、新手、联动（常规统计期次不含联动跃迁）与“全部”（新手以外的全部跃迁）各合为一栏；“集录”“抽奖”“喵喵”等词不指定卡池；每张图最多 10 期，更早期次在页尾说明，顶部统计包含全部数据；写卡池再加“全部”（如“角色全部统计”）时每 10 期一张合并转发；上游在 #星铁 前缀下误将“全部统计”当作分页的抽卡统计，这里按其本意，单写“全部”仍为全部统计；UP 判定与分期统计同上游 `apps/gacha/GachaData.js`，期次取自同一卡池资料，最后一期之后的“新版本”不设结束日期（上游写死的结束日期已过），落在各期之外的抽取之后更早的抽取仍按各自的期次统计（上游此后不再查找期次）；页头头像上游按展柜的 headIcon 查找角色，星铁的头像编号不是角色编号，因此总是显示默认的卡芙卡，这里照做。`templates/character-card/` 按 miao-plugin 的 character/character-card 改写（MIT），横竖版的页面宽度与缩放同上游 `apps/character/AvatarCard.js`，触发写法同其 check；上游按角色的元素名给技能图标分类，没有与之对应的星铁样式，图标沿用页面默认的元素底色，这里照做。`templates/note/`、`templates/card/`、`templates/gacha/`、`templates/challenge/`、`templates/challenge-all/`、`templates/peak/`、`templates/peak-recent/`、`templates/rogue/`、`templates/swarm/`、`templates/divergent/`、`templates/divergent-run/`、`templates/currency-wars/` 与 `templates/currency-wars-archive/` 按 StarRail-plugin 的 new_note、card/card、gatcha/new、challenge/index、index_all、index_peak、peak_recent、rogue/rogue、rogue/rogue_locust、rogue/rogueTourn、rogueTournNormal/rogueTournWeek 与 gridFight/gridFight、gridFightArchive 改写（卡片同上游 apps/hkrpg.js 的 card，官方未设手机主题时换成上游指定的主题；上游的卡片规则没有结尾限定，会抢走“角色分析”“角色面板”等以卡片、探索、角色开头的写法，这里只接这三个词与紧随的 UID；“角色”在上游由这条规则先匹配，miao 的角色列表规则不接星铁前缀，因此同样出卡片；常规演算、周期演算与货币战绩同上游 utils/archiveArgs.js 读取数字下标与 UID，超出时同上游回复；上游在伤害记录找不到对应角色或羁绊时无法出图，这里只空出该项；常规演算与周期演算两页的样式只差周期信息一段，共用 rogueTournWeek 的样式；上游按 HTML 原样输出的面具说明与周期效果，这里只保留换行、颜色与加粗，其余标签转义或丢弃；黄金与机械与不可知域上游没有出图，仍以文字回复）（Apache-2.0，见 `LICENSES/StarRail-plugin-Apache-2.0.txt`），跃迁记录页的统计同上游 GatchaData.stat，回答以“记录”“分析”或“祈愿”结尾的写法；总览含角色联动与光锥联动跃迁，“联动”同上游只指角色联动跃迁，光锥联动写“光锥联动”，联动页头只列该跃迁的未出五星抽数。常规跃迁按 miao 固定快照的卡池资料分组，标题写卡池名（直接取自 miao 的 poolNameSr）；两期维护空窗内的记录归入前一期，距其结束超过 24 小时的不归组。联动跃迁按 tools/collab-pools.json 的两期 Fate 池归组，取同类型开池最晚且早于抽取时间的一期（结束时间同上游用 2036 年哨兵值）。持有率分母为资料库中四星、五星角色与光锥的数量，与上游 baseData 同样不计开拓者，强化角色与原角色算一个，固定提交时两者都是 233。发送抽卡链接后的记录图为这一页的总览；`templates/help/` 按本插件的群青星空配色绘制，页头沿用 StarRail-plugin help/index 默认主题的插画（Apache-2.0，见 `LICENSES/StarRail-plugin-Apache-2.0.txt`）。`templates/ledger-count/` 按 Miao-Yunzai 原神插件的 html/ledger/ledger-count-sr 改写（StarRail-plugin 没有星琼统计页；GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），柱状图与环形图上游用 G2Plot 绘制，这里按相同的内边距、刻度算法、半径与标签规则输出 SVG；页头的星铁标志来自该插件仓库，素材来源为“Yunzai 页面素材”，只取 `resources/StarRail/img/other/`、星琼页用到的 `resources/StarRail/img/note/` 与 `resources/font/HYWenHei-55W.ttf`、uid 页用到的 `resources/img/icon/`，以及公告页用到的 `resources/html/mysNews/`、`resources/html/mysNews-list/`、`resources/font/tttgbnumber.ttf`。`templates/ledger/` 按同一上游的 html/ledger/ledger-sr 改写（GPL-3.0），月份的读取同 `model/ledger.js` 的 getMonth，数额与抽数同 dealData，环形图同样输出 SVG；genshinLayout 的横幅样式只作用于页头，这一页没有页头。getMonth 在一、二月把上一年的十一、十二月改成本月（判断后来月份的条件 `month <= 9 + month` 恒成立，星铁的 YYYYMM 月份又会记到今年），这里按上一年的月份查询。`templates/calendar/` 与 `templates/calendar-list/` 按 miao-plugin 的 `wiki/calendar` 星铁模式改写（MIT；列表模式在上游缩窄页面，这里以第二份模板取其宽度），公告时间、跃迁图片、三种挑战的 42 天轮换与异相仲裁周期、排序合并的算法同上游 `apps/wiki/CalendarSr.js`。上游锚点在 2026 年三种挑战各缩短一期后不再对应，这里改从 2026-10-05（末日幻影）、2026-10-19（虚构叙事）、2026-11-02（混沌回忆）起算，并按向前取整计算（上游在锚点晚于本周时算错）；上游把轮换条目的类型写成函数，导致样式与图标缺失，这里按其所列类型出图。`templates/character-wiki/` 按 miao-plugin 的 `wiki/character-wiki` 星铁模式改写（MIT），角色图鉴与资料展示命途、属性、归属、CV、80 级基础属性和按上游键序排列的培养材料，材料图标来自 `meta-sr/material`。`templates/character-talent/` 按 miao-plugin 的 `wiki/character-talent` 星铁模式与 `common/tpl/talent-detail` 改写（MIT），技能、行迹与星魂资料读取 miao 素材来源的 `meta-sr/character/<角色>/data.json`（为此 miao 素材来源也保留 `.json` 文件），描述中的数值按上游 getDesc 在 8 级（普攻与忆灵技能 5 级）代入，数字格式同 JavaScript 的 toFixed；描述 HTML 只保留标题、换行、斜体、加粗、nobr、span 与颜色。`templates/panel/` 的样式由 miao-plugin 固定提交的 `common/common.css` 与 `character/profile-detail.css` 转换而来，`templates/training/` 按 miao 的 `character/profile-stat` 与通用角色卡片改写（MIT）。两者的图片地址都改为宿主渲染资源。模板用到的图片、字体与资料文件取自各上游的固定提交，由 `scripts/bundle-artwork.py` 复制到插件包的 `assets/` 随插件分发（许可同各上游，见 `LICENSES/`）；管理员可通过“素材更新”从上游仓库下载新版本，下载的文件优先使用。深渊战绩里的角色、增益与首领图标是官方图片地址，首次出图时按需缓存到“官方图片缓存”。

`templates/uid-list/` 按 Miao-Yunzai 原神插件的 html/user/uid-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），页面框架与样式沿用 miao-plugin 的 common/layout/elem 与 common/common.css（MIT，见 `LICENSES/miao-plugin-MIT.txt`），同 miao 的 1.4 倍缩放；图片地址改为宿主渲染资源。玩家的头像与名片按 miao 的 faceImgs 取该 UID 已保存面板中编号最小的角色，没有时用 miao 的通用头像与名片。

`templates/rank/` 按 miao-plugin 的 character/rank-profile-list 星铁模式改写（MIT），容器宽度同上游在页面内设置，群排行按 ark-plugin 改写后的宽度 930（欢愉命途 960、记忆命途 1000），并同其备份的 `miao-plugin-rank/resources/character/rank-profile-list.html` 加上全服排名一栏（再宽 180）；ark-plugin 的自定义排行沿用这一页面（MIT，见 `LICENSES/ark-plugin-MIT.txt`），同 `apps/customRank.js` 写入“全服数据”、排序与筛选说明及请求行数，页面宽 930（欢愉命途 960、记忆命途 1000），评级按六件遗器的平均分阈值（56 分以上为 MAX）。

`templates/panel/` 另按 ark-plugin 备份的 `miao-plugin-rank/resources/character/profile-detail.html`（MIT，见 `LICENSES/ark-plugin-MIT.txt`）加入面板排名：名次行接在伤害表后，排名统计块照搬其布局与内联样式（不写上游指向排名统计命令的提示，本插件聊天不提供该命令），两张图按其 ECharts 选项（默认网格、smooth、面积渐变与标注）输出为 SVG，换装面板的伤害变化同其伤害表的写法（上游的内联样式改为同值的类）；渐变的色标按上游的原顺序输出（上游把红色色标写在前一色标之前，浏览器因此从本面板的位置起画 2.5 个百分点的红带，这里保持一致）；上游在星铁中同样写“圣遗物排名”“圣遗物评分”，这里按星铁的称呼写“遗器排名”“遗器评分”。

`templates/atlas-index/` 按 Atlas 的 resource/massage/text.html 与 text.css 改写（GPL-3.0，见 `LICENSES/Atlas-GPL-3.0.txt`），同上游的 1.5 倍缩放，数字字体取自 Yunzai 的 tttgbnumber；上游的 Source Han Sans CN Bold 字体超过宿主单个渲染资源 16 MiB 的上限，不随附，其余文字使用渲染器的字体。

`templates/charge/` 与 `templates/online/` 按 StarRail-plugin 的 charge 与 online 页面改写（Apache-2.0），“星琼记录”等与“在线时长”同上游 apps/hkrpg.js：资产记录读取官方自助查询的一页（上游按每页 10 条请求，官方现在忽略页长与页码，固定每页 20 条并按上一页最后一条记录的编号翻页，因此每页为 20 条，第 N 页由账号插件按编号依次翻到）默认只保留获取的记录（开拓力只保留多于 1 点的），写“消耗”只保留减少的记录，写“全部”两者都保留（开拓力略去 1 点的自然回复），表头随之写获取、消耗或变化数量；上游的“光锥记录”“武器记录”被跃迁记录的规则先匹配而从不出图，这里同样交给跃迁记录，光锥的资产记录须写“获取”“消耗”或“全部”（如“光锥获取记录”）。在线时长读取开拓力记录中的“随时间回复开拓力”，按上游 statisticsOnlineDateGeneral 与 statisticOnlinePeriods 把每次单独回复的 1 点记为之前 6 分钟在线，逐日统计小时数并画出近七天的在线时段（上游逐分钟扫描使每段少记一分钟，这里照做；上游把零点后六分钟内的记录算到前一天时取错了日期，这里按其本意记到前一天）；按记录编号逐页读取到没有更多记录（上游按页码翻页，官方已不再按页码推进），为在事件时限内回复最多读取 40 秒。两张 ECharts 图（近七天柱状图与日历热力图）按 ECharts 的默认布局、配色与分段输出 SVG，热力图上游固定为 2023 年，这里显示当前年份。

收益曲线同 StarRail-plugin 的 apps/srxx.js，直接发送该仓库 `resources/srsr/sy/` 中的属性收益曲线图（Apache-2.0，见 `LICENSES/StarRail-plugin-Apache-2.0.txt`；上游在文件头注明各图作者），与上游一样以合并转发发送，平台不支持合并转发时改为一条消息；曲线是词条分配的理论计算，不随版本变化，取自“星铁插件图片”素材来源。同一上游文件的参考面板、强度榜、全角色攻略、深渊攻略与商店光锥推荐停在 1.x 版本，不收录。面板帮助是本插件自写的文字说明。

攻略按 StarRail-plugin 的 apps/strategy.js 读取米游社攻略合集（`getPostFullInCollection`），只发送官方图床的图片地址，不下载、不随包分发作者图片，内容与观点归原作者所有。上游的来源（初始镜像 1996095、小橙子阿 1998643、星穹中心 2029394/2009142/2038092）在 2024-03 至 2025-12 间先后停更或清空，2026-10 改用仍在更新、覆盖 4.x 角色的合集：1. HoYo青枫「崩坏：星穹铁道丨养成攻略集」（1998324，默认，完整一图流长图）；2. 让我摸个鱼吧Moyu「《摸鱼笔记》系列大合集」（2259414）；3. 祈鸢ya「星铁角色攻略」（2049699）。标题匹配规则见 contracts/game-v1.md。

素材图按 Yunzai 原神插件 apps/material.js 的星铁分支读取米游社合集，只发送官方图床地址；Yunzai 依次读取小橙子阿 1998643（停在 V3.8）、紫喵Azunya 2146693（停在 V3.2）、让我摸个鱼吧Moyu 2279356；2026-10 起先读仍在更新的 2279356（已收录到真珠），再读另两个；三个合集分别取的图片位置与 1998643 的限制同契约；标题匹配见 contracts/game-v1.md。

`templates/news/` 与 `templates/news-list/` 按 Yunzai 原神插件的 html/mysNews 与 html/mysNews-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），公告、资讯、活动、米游社搜索、帖子与预估按其 mysNews 的规则出图。米游社搜索同上游 mysSearch，消息含“签到”时不处理；“米游社签到”“mys签到”“米游社星铁签到”由社区签到接收；详情页的样式去掉了米游社编辑器、加载、提示与回复控件等页面用不到的规则。米游社正文是任意 HTML，而渲染器可以联网，因此正文按页面样式用到的标签、类名与颜色字号等样式重建，图片只经渲染资源引用（官方图片缓存），链接与脚本不保留。与上游不同之处：正文中没有地址的超链图片按帖子的 structured_content 补上（上游显示为空白）；纯图片帖显示全部图片（上游只显示最后一张）；上游按 4000 像素分段截图，这里出一整张长图；二维码由 go-qrcode 生成（MIT，见 `LICENSES/go-qrcode-MIT.txt`），指向帖子所在游戏的米游社地址（上游固定为原神路径）。页面的米游社标志、图标字体、列表页背景与数字字体取自“Yunzai 页面素材”素材来源。星铁的“预估”同 StarRail-plugin 的 srEstimate，合并转发作者最新“星琼统计”帖的图片；“星琼预估”“盘点”等写法同 Yunzai 原神插件，按“星琼统计汇总”出图。

## 有意不跟进的上游功能

StarRail-plugin 的自带面板与伤害引擎不移植，统一使用 miao；不提供 `同步面板`、`同步抽卡记录`（本插件只有一份业务档案）、alias.js（别名来自 miao）、PanelPic 图片包的 gitee/github 切换后缀，以及上游帮助和更新日志页面（命令帮助由 info.json 提供）。周期演算仍默认本期。

miao 的 Tomato gacha-detail 欧皇度、大小保底与“全部记录”不移植，跃迁记录页继续使用 StarRail-plugin；不移植 proShareProfileImg、多图库来源配置、guoba、TRSS 重启、原神资源与面板图标 CSS、artis-mark 页面。面板换装只接受“换”，与 miao 837a303858、ark bdced6bf 的规则相同。
