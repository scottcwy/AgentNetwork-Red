package bd

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type wrapCard struct {
	SeedID       string
	ProfileURL   string
	CoverPhoto   string
	SourceEvent  string
	CapturedAt   string
	Collector    string
	DisplayName  string
	TeamName     string
	TeamScope    string
	Note         string
	DisplayTitle string
}

type wrapPage struct {
	GeneratedAt string
	SeedCount   int
	Events      []string
	Cards       []wrapCard
}

func GenerateWrapUp(root, outputPath string) error {
	seeds, err := LoadSeeds(root)
	if err != nil {
		return err
	}
	if issues := ValidateSeeds(root, seeds); len(issues) > 0 {
		return fmt.Errorf("bd validation failed:\n- %s", joinIssues(issues))
	}

	SortSeedsByCapturedAtDesc(seeds)

	repoRoot := filepath.Dir(root)
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	eventSet := make(map[string]struct{})
	cards := make([]wrapCard, 0, len(seeds))
	for _, seed := range seeds {
		eventSet[seed.SourceEvent] = struct{}{}

		absolutePhoto, err := resolveCoverPhotoPath(repoRoot, seed.CoverPhoto)
		if err != nil {
			return err
		}
		relativePhoto, err := filepath.Rel(outputDir, absolutePhoto)
		if err != nil {
			return err
		}

		card := wrapCard{
			SeedID:       seed.SeedID,
			ProfileURL:   seed.XHSProfileURL,
			CoverPhoto:   filepath.ToSlash(relativePhoto),
			SourceEvent:  seed.SourceEvent,
			CapturedAt:   formatCapturedAt(seed.CapturedAt),
			Collector:    seed.Collector,
			DisplayName:  seed.DisplayName,
			TeamName:     seed.TeamName,
			TeamScope:    seed.TeamScope,
			Note:         seed.Note,
			DisplayTitle: seed.DisplayName,
		}
		if card.DisplayTitle == "" {
			card.DisplayTitle = seed.SeedID
		}
		cards = append(cards, card)
	}

	events := make([]string, 0, len(eventSet))
	for event := range eventSet {
		events = append(events, event)
	}
	sort.Strings(events)

	page := wrapPage{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SeedCount:   len(seeds),
		Events:      events,
		Cards:       cards,
	}

	var rendered bytes.Buffer
	if err := template.Must(template.New("wrap").Parse(wrapHTML)).Execute(&rendered, page); err != nil {
		return err
	}

	return os.WriteFile(outputPath, rendered.Bytes(), 0o644)
}

func joinIssues(issues []string) string {
	result := ""
	for i, issue := range issues {
		if i > 0 {
			result += "\n- "
		}
		result += issue
	}
	return result
}

func formatCapturedAt(raw string) string {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return parsed.Format("2006-01-02 15:04 UTC")
}

const wrapHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>XHS Seed Wrap-Up</title>
  <style>
    :root {
      --bg: #f7f0e8;
      --panel: #fffaf5;
      --ink: #2d231a;
      --muted: #705f4f;
      --line: #ddcdbb;
      --accent: #c95f35;
      --accent-soft: #f7d8c8;
      --shadow: 0 20px 40px rgba(63, 32, 16, 0.08);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Helvetica Neue", "PingFang SC", "Noto Sans SC", sans-serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, #fff6cf 0, transparent 32%),
        linear-gradient(180deg, #fdf8f3 0%, var(--bg) 100%);
    }
    .shell {
      max-width: 1180px;
      margin: 0 auto;
      padding: 48px 24px 72px;
    }
    .hero {
      padding: 28px 32px;
      border: 1px solid var(--line);
      border-radius: 28px;
      background: rgba(255, 250, 245, 0.92);
      box-shadow: var(--shadow);
    }
    .eyebrow {
      display: inline-block;
      margin-bottom: 12px;
      padding: 8px 12px;
      border-radius: 999px;
      background: var(--accent-soft);
      color: var(--accent);
      font-size: 13px;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    h1 {
      margin: 0 0 12px;
      font-size: clamp(34px, 5vw, 58px);
      line-height: 0.96;
    }
    .hero p {
      max-width: 780px;
      margin: 0;
      color: var(--muted);
      font-size: 18px;
      line-height: 1.6;
    }
    .meta {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      margin-top: 22px;
    }
    .meta span,
    .events li {
      padding: 10px 14px;
      border: 1px solid var(--line);
      border-radius: 999px;
      background: #fff;
      color: var(--muted);
      font-size: 14px;
    }
    .events {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      padding: 0;
      margin: 22px 0 0;
      list-style: none;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 18px;
      margin-top: 28px;
    }
    .card {
      overflow: hidden;
      border: 1px solid var(--line);
      border-radius: 24px;
      background: var(--panel);
      box-shadow: var(--shadow);
    }
    .card img {
      display: block;
      width: 100%;
      aspect-ratio: 4 / 3;
      object-fit: cover;
      background: #f1e5d9;
    }
    .copy {
      padding: 20px;
    }
    .copy h2 {
      margin: 0 0 6px;
      font-size: 24px;
    }
    .subhead {
      margin: 0 0 12px;
      color: var(--accent);
      font-size: 14px;
      font-weight: 700;
      letter-spacing: 0.04em;
      text-transform: uppercase;
    }
    .copy p {
      margin: 0 0 10px;
      color: var(--muted);
      line-height: 1.55;
      font-size: 15px;
    }
    .copy a {
      display: inline-flex;
      margin-top: 8px;
      color: var(--accent);
      text-decoration: none;
      font-weight: 700;
    }
    .copy a:hover { text-decoration: underline; }
  </style>
</head>
<body>
  <main class="shell">
    <section class="hero">
      <div class="eyebrow">Red Agent Network BD Wrap-Up</div>
      <h1>小红书种子库 Wrap-Up Display</h1>
      <p>这个页面只消费 GitHub 里的小红书种子和封面图，用来做项目收尾展示，而不是运营后台。可选备注会在这里展示，缺省采集者统一按 Zeena 处理。</p>
      <div class="meta">
        <span>Seeds: {{.SeedCount}}</span>
        <span>Events: {{len .Events}}</span>
        <span>Generated: {{.GeneratedAt}}</span>
      </div>
      <ul class="events">
        {{range .Events}}<li>{{.}}</li>{{end}}
      </ul>
    </section>

    <section class="grid">
      {{range .Cards}}
      <article class="card">
        <img src="{{.CoverPhoto}}" alt="{{.DisplayTitle}}">
        <div class="copy">
          <p class="subhead">{{.SourceEvent}}</p>
          <h2>{{.DisplayTitle}}</h2>
          {{if .TeamName}}<p><strong>团队：</strong>{{.TeamName}}</p>{{end}}
          {{if .TeamScope}}<p>{{.TeamScope}}</p>{{end}}
          {{if .Note}}<p>{{.Note}}</p>{{end}}
          <p><strong>采集时间：</strong>{{.CapturedAt}}</p>
          <p><strong>采集者：</strong>{{.Collector}}</p>
          <a href="{{.ProfileURL}}" target="_blank" rel="noreferrer">打开小红书主页</a>
        </div>
      </article>
      {{end}}
    </section>
  </main>
</body>
</html>
`
