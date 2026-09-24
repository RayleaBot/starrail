package app

import (
	"context"
	"regexp"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// cloudQQPattern is a QQ number, which ark binds a verified UID to.
var cloudQQPattern = regexp.MustCompile(`^[1-9][0-9]{4,11}$`)

// arkVerifyCommand is ark-plugin's ark绑定星铁uid (arkGetBindUid), which
// asks ark for the code to set as the game signature, and ark验证星铁uid
// (arkBindUid), which has ark check the signature and bind the UID to the
// sender's QQ. The UID is the one miao's getTargetUid gives: a mentioned
// player's, else the sender's in use. ark binds QQ numbers, which only
// OneBot11 senders carry, so others cannot verify.
func (a *App) arkVerifyCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if len(args) > 0 {
		// Upstream's rules end with the command words.
		return event.Result(map[string]any{"handled": false})
	}
	route, body := "verify/code", map[string]any{"type": arkGame}
	if command == "cloud-verify" {
		if event.Event.SourceProtocol != "onebot11" || !cloudQQPattern.MatchString(event.Event.Actor.ID) {
			return event.SendText("ark验证将uid与QQ号绑定，当前聊天平台无法验证")
		}
		route, body["qq"] = "verify", event.Event.Actor.ID
	}
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil || owner.UID == "" {
		return event.SendText("请先绑定UID")
	}
	body["uid"] = owner.UID
	decoded, err := a.arkRequest(ctx, event, route, body)
	return event.SendText(arkVerifyReply(a.Game.Prefix, route, decoded, err))
}

// arkVerifyReply is what ark-plugin replies to a verify route's answer: the
// verification code and how to use it, or 验证成功, else the ERROR_MAP text
// of ark's retcode. A request that fails has no retcode and reads 未知错误.
// The code's instructions keep upstream's Star Rail steps.
func arkVerifyReply(prefix, route string, decoded any, err error) string {
	result := asObject(decoded)
	code := asText(result["retcode"])
	switch {
	case err != nil:
		return "未知错误"
	case code != "100":
		return cmpOr(arkErrors(prefix)[code], "未知错误")
	case route == "verify":
		return "验证成功"
	}
	return "验证码: " + asText(asObject(result["data"])["verifyCode"]) + "\n使用方式：\n星铁：手机——右上角三点——漫游签证——设置签名——填入验证码，5-10分钟后输入 " + prefix + "ark验证星铁uid\n验证码有效期24小时，验证通过后自动与QQ绑定，在其他Bot上无需再次绑定"
}

// arkErrors is ark-plugin's ERROR_MAP by retcode, with 304 naming this
// plugin's verify command.
func arkErrors(prefix string) map[string]string {
	return map[string]string{
		"-1":  "插件版本过低，请更新插件",
		"101": "角色ID不存在",
		"102": "未查询到角色信息",
		"103": "请求参数错误",
		"104": "请求超过速率限制",
		"105": "未知错误",
		"106": "数据过大，请确保导出的数据小于2MB",
		"201": "请求超过速率限制，请5分钟后重试",
		"202": "未发现该用户的数据，请重新导出面板",
		"301": "请求类型仅支持原神/星铁",
		"302": "验证失败，个人签名不匹配，请五分钟后重试",
		"303": "验证失败，请稍后再试",
		"304": "该uid未验证号主，请通过 " + prefix + "ark验证星铁uid 验证uid",
		"305": "验证超时，请重新绑定",
		"306": "验证失败，未获取到签名，请五分钟后重试",
		"307": "服务器中无该uid数据...",
	}
}
