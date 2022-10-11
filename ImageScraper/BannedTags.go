package ImageScraper

/***
TAGS BELOW ARE NSFW AND MAY BE OFFENSIVE TO SOME USERS
*/

/**
SCROLL DOWN
*/

var BannedTags = []string{
	"assisted_rape",
	"kamado_nezuko",
	"kanna_kamui_(dragon)_(maidragon)",
	"young_girl",
	"young_boy",
	"child",
	"young",
	"megumin",
	"kanna_kamui",
	"harvin",
	"pokemon_(game)",
	"paimon_(genshin_impact)",
	"shoujo",
	"guro",
	"hanging",
	"execution",
	"karyl_(princess_connect!)",
	"girls_und_panzer",
	"undertale",
	"hataraku_saibou",
	"gawr_gura",
	"z1_leberecht_maass_(azur_lane)",
	"kemono_friends",
	"ebine_(flower_knight_girl)",
	"cub",
	"kokkoro_(princess_connect!)",
	"loli",
	"shota",
	"lolicon",
	"shota",
	"shotacon",
	"underage",
	"scat",
	"vomit",
	"rape",
	"watersports",
	"vore",
	"anya_(spy_x_family)",
}

func TagIsBanned(tag string) bool {
	for _, bannedTag := range BannedTags {
		if tag == bannedTag {
			return true
		}
	}
	return false
}
