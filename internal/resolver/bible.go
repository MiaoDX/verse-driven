package resolver

// bibleBook is one row of the Bible book alias table.
type bibleBook struct {
	canonical string   // canonical English name, e.g. "John", "1 John"
	aliases   []string // lowercased aliases (English + Simplified Chinese)
}

// bibleBooks is the Bible alias table. All English aliases must be lowercase;
// Chinese aliases are matched verbatim. Order is irrelevant — buildBibleIndex
// sorts by alias length descending to ensure the longest match wins
// (so "1 john" beats "john" on input "1 john 3:16").
var bibleBooks = []bibleBook{
	// Old Testament
	{"Genesis", []string{"genesis", "gen", "gn", "ge", "创世记", "创"}},
	{"Exodus", []string{"exodus", "exod", "exo", "ex", "出埃及记", "出"}},
	{"Leviticus", []string{"leviticus", "lev", "lv", "利未记", "利"}},
	{"Numbers", []string{"numbers", "num", "nm", "民数记", "民"}},
	{"Deuteronomy", []string{"deuteronomy", "deut", "dt", "申命记", "申"}},
	{"Joshua", []string{"joshua", "josh", "jos", "约书亚记", "书"}},
	{"Judges", []string{"judges", "judg", "jdg", "士师记", "士"}},
	{"Ruth", []string{"ruth", "rth", "ru", "路得记", "得"}},
	{"1 Samuel", []string{"1 samuel", "1samuel", "1 sam", "1sam", "1 sa", "1sa", "i samuel", "撒母耳记上", "撒上"}},
	{"2 Samuel", []string{"2 samuel", "2samuel", "2 sam", "2sam", "2 sa", "2sa", "ii samuel", "撒母耳记下", "撒下"}},
	{"1 Kings", []string{"1 kings", "1kings", "1 kgs", "1kgs", "1 ki", "1ki", "i kings", "列王纪上", "王上"}},
	{"2 Kings", []string{"2 kings", "2kings", "2 kgs", "2kgs", "2 ki", "2ki", "ii kings", "列王纪下", "王下"}},
	{"1 Chronicles", []string{"1 chronicles", "1chronicles", "1 chron", "1chron", "1 chr", "1chr", "i chronicles", "历代志上", "代上"}},
	{"2 Chronicles", []string{"2 chronicles", "2chronicles", "2 chron", "2chron", "2 chr", "2chr", "ii chronicles", "历代志下", "代下"}},
	{"Ezra", []string{"ezra", "ezr", "以斯拉记", "拉"}},
	{"Nehemiah", []string{"nehemiah", "neh", "ne", "尼希米记", "尼"}},
	{"Esther", []string{"esther", "esth", "est", "以斯帖记", "斯"}},
	{"Job", []string{"job", "jb", "约伯记", "伯"}},
	{"Psalms", []string{"psalms", "psalm", "psa", "ps", "pss", "诗篇", "诗"}},
	{"Proverbs", []string{"proverbs", "prov", "pr", "箴言", "箴"}},
	{"Ecclesiastes", []string{"ecclesiastes", "eccl", "ec", "传道书", "传"}},
	{"Song of Solomon", []string{"song of solomon", "song of songs", "song", "sos", "雅歌", "歌"}},
	{"Isaiah", []string{"isaiah", "isa", "is", "以赛亚书", "赛"}},
	{"Jeremiah", []string{"jeremiah", "jer", "je", "耶利米书", "耶"}},
	{"Lamentations", []string{"lamentations", "lam", "la", "耶利米哀歌", "哀"}},
	{"Ezekiel", []string{"ezekiel", "ezek", "ezk", "以西结书", "结"}},
	{"Daniel", []string{"daniel", "dan", "da", "但以理书", "但"}},
	{"Hosea", []string{"hosea", "hos", "ho", "何西阿书", "何"}},
	{"Joel", []string{"joel", "joe", "jl", "约珥书", "珥"}},
	{"Amos", []string{"amos", "amo", "am", "阿摩司书", "摩"}},
	{"Obadiah", []string{"obadiah", "obad", "ob", "俄巴底亚书", "俄"}},
	{"Jonah", []string{"jonah", "jon", "jnh", "约拿书", "拿"}},
	{"Micah", []string{"micah", "mic", "mi", "弥迦书", "弥"}},
	{"Nahum", []string{"nahum", "nah", "na", "那鸿书", "鸿"}},
	{"Habakkuk", []string{"habakkuk", "hab", "hb", "哈巴谷书", "哈"}},
	{"Zephaniah", []string{"zephaniah", "zeph", "zep", "西番雅书", "番"}},
	{"Haggai", []string{"haggai", "hag", "hg", "哈该书", "该"}},
	{"Zechariah", []string{"zechariah", "zech", "zec", "撒迦利亚书", "亚"}},
	{"Malachi", []string{"malachi", "mal", "ml", "玛拉基书", "玛"}},

	// New Testament
	{"Matthew", []string{"matthew", "matt", "mt", "马太福音", "太"}},
	{"Mark", []string{"mark", "mrk", "mk", "马可福音", "可"}},
	{"Luke", []string{"luke", "luk", "lk", "路加福音", "路"}},
	{"John", []string{"john", "joh", "jn", "约翰福音", "约"}},
	{"Acts", []string{"acts", "act", "ac", "使徒行传", "徒"}},
	{"Romans", []string{"romans", "rom", "ro", "罗马书", "罗"}},
	{"1 Corinthians", []string{"1 corinthians", "1corinthians", "1 cor", "1cor", "1 co", "1co", "i corinthians", "哥林多前书", "林前"}},
	{"2 Corinthians", []string{"2 corinthians", "2corinthians", "2 cor", "2cor", "2 co", "2co", "ii corinthians", "哥林多后书", "林后"}},
	{"Galatians", []string{"galatians", "gal", "ga", "加拉太书", "加"}},
	{"Ephesians", []string{"ephesians", "eph", "以弗所书", "弗"}},
	{"Philippians", []string{"philippians", "phil", "php", "腓立比书", "腓"}},
	{"Colossians", []string{"colossians", "col", "歌罗西书", "西"}},
	{"1 Thessalonians", []string{"1 thessalonians", "1thessalonians", "1 thess", "1thess", "1 th", "1th", "i thessalonians", "帖撒罗尼迦前书", "帖前"}},
	{"2 Thessalonians", []string{"2 thessalonians", "2thessalonians", "2 thess", "2thess", "2 th", "2th", "ii thessalonians", "帖撒罗尼迦后书", "帖后"}},
	{"1 Timothy", []string{"1 timothy", "1timothy", "1 tim", "1tim", "1 ti", "1ti", "i timothy", "提摩太前书", "提前"}},
	{"2 Timothy", []string{"2 timothy", "2timothy", "2 tim", "2tim", "2 ti", "2ti", "ii timothy", "提摩太后书", "提后"}},
	{"Titus", []string{"titus", "tit", "ti", "提多书", "多"}},
	{"Philemon", []string{"philemon", "phlm", "phm", "腓利门书", "门"}},
	{"Hebrews", []string{"hebrews", "heb", "希伯来书", "来"}},
	{"James", []string{"james", "jas", "jm", "雅各书", "雅"}},
	{"1 Peter", []string{"1 peter", "1peter", "1 pet", "1pet", "1 pe", "1pe", "i peter", "彼得前书", "彼前"}},
	{"2 Peter", []string{"2 peter", "2peter", "2 pet", "2pet", "2 pe", "2pe", "ii peter", "彼得后书", "彼后"}},
	{"1 John", []string{"1 john", "1john", "1 jn", "1jn", "i john", "约翰一书", "约一"}},
	{"2 John", []string{"2 john", "2john", "2 jn", "2jn", "ii john", "约翰二书", "约二"}},
	{"3 John", []string{"3 john", "3john", "3 jn", "3jn", "iii john", "约翰三书", "约三"}},
	{"Jude", []string{"jude", "jud", "jd", "犹大书", "犹"}},
	{"Revelation", []string{"revelation", "rev", "re", "启示录", "启"}},
}
