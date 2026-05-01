package resolver

// quranSurah maps a surah's English/Arabic name to its number.
type quranSurah struct {
	number  int
	aliases []string
}

// quranSurahs lists a representative subset of surahs by name. Lookup by
// number (e.g. "Quran 2:255") is handled separately in the resolver.
var quranSurahs = []quranSurah{
	{1, []string{"al-fatihah", "al fatihah", "fatihah", "al-faatihah"}},
	{2, []string{"al-baqarah", "al baqarah", "baqarah", "al-baqara"}},
	{3, []string{"aal-imran", "ali imran", "al imran", "imran", "aal imran"}},
	{4, []string{"an-nisa", "an nisa", "nisa", "al-nisa"}},
	{5, []string{"al-maidah", "al maidah", "maidah"}},
	{18, []string{"al-kahf", "al kahf", "kahf"}},
	{36, []string{"yasin", "ya-sin", "ya sin", "yaseen"}},
	{55, []string{"ar-rahman", "ar rahman", "rahman"}},
	{67, []string{"al-mulk", "al mulk", "mulk"}},
	{112, []string{"al-ikhlas", "al ikhlas", "ikhlas"}},
	{113, []string{"al-falaq", "al falaq", "falaq"}},
	{114, []string{"an-nas", "an nas", "nas"}},
}
