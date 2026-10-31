// Package notice は、友だちへ送る通知の内容を表す。
package notice

import (
	"strings"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

// namespace は、通知のキー(UUID v5)を作るための、このアプリ固有の名前空間。
var namespace = mustParseUUID("7d0b6f0e-3c1a-4f55-9d3e-2a8f6c1b9e44")

// Notice は送信する通知。送信先(誰に送るか)は持たない。
type Notice struct {
	// Key は、同じ通知を表す安定した UUID。送信先のサービスが重複を判定するために使う。
	Key string
	// Text は本文。
	Text string
}

// StartingSoon は、プログラムがまもなく始まることを伝える通知を作る(MVP は固定の文面)。
func StartingSoon(p *program.Program) Notice {
	var b strings.Builder
	if p.EventName() != "" {
		b.WriteString("【" + p.EventName() + "】\n")
	}
	b.WriteString("まもなく開始します\n")
	b.WriteString(p.Name() + "\n")
	b.WriteString("開始時刻: " + p.StartAt().Format("15:04"))
	if p.Location() != "" {
		b.WriteString("\n場所: " + p.Location())
	}
	return Notice{
		Key:  uuidV5(namespace, "starting-soon:"+string(p.ID())),
		Text: b.String(),
	}
}
