package main

import "sort"

// ---------------------------------------------------------------------------
// SECTIONS
//
// A section is one of the big rooms of PathFinder (Programming, Entertainment,
// Learning, Lifestyle, Create & Work). Each one has its own dark colour, its
// own topics, and the topics hold the 15 places you can be redirected to.
// ---------------------------------------------------------------------------

// Fork is the big choice card that sits at the top of a section page.
// Programming uses it: watch documentaries first, or go straight to learning.
type Fork struct {
	Title string
	Blurb string
	Slug  string // the topic it opens
	Href  string // where the card goes, when it is not a topic
}

type Section struct {
	Slug    string
	Name    string
	Tagline string
	Blurb   string
	Theme   string // css theme key: prog, ent, learn, life, work
	Forks   []Fork
	Topics  []Topic
	// Direct shows the places themselves on the section page, with no
	// topic step in between (Read Manga, Read Manhwa).
	Direct bool
}

// res is just a short way to write a Resource in the lists below.
func res(id, name, url, description string) Resource {
	return Resource{ID: id, Name: name, URL: url, Description: description}
}

// Extra topics that are not programming languages.
var otherTopics = map[string]Topic{
	"documentaries": {
		Slug:  "documentaries",
		Name:  "Documentaries",
		Blurb: "15 films and channels about the people who built this world",
		Resources: []Resource{
			res("doc-imitation-game", "The Imitation Game", "https://www.imdb.com/title/tt2084970/", "Alan Turing breaks Enigma"),
			res("doc-social-network", "The Social Network", "https://www.imdb.com/title/tt1285016/", "How Facebook started"),
			res("doc-hidden-figures", "Hidden Figures", "https://www.imdb.com/title/tt4846340/", "The women who put NASA in orbit"),
			res("doc-steve-jobs", "Steve Jobs", "https://www.imdb.com/title/tt2080374/", "Three product launches, one man"),
			res("doc-pirates", "Pirates of Silicon Valley", "https://www.imdb.com/title/tt0168122/", "Apple and Microsoft, the early years"),
			res("doc-alphago", "AlphaGo", "https://www.imdb.com/title/tt6700846/", "The match that changed AI"),
			res("doc-general-magic", "General Magic", "https://www.imdb.com/title/tt6849786/", "The phone that came 20 years early"),
			res("doc-lo-and-behold", "Lo and Behold", "https://www.imdb.com/title/tt5275828/", "Werner Herzog on the internet"),
			res("doc-great-hack", "The Great Hack", "https://www.imdb.com/title/tt9358209/", "What happened to your data"),
			res("doc-halt-catch-fire", "Halt and Catch Fire", "https://www.imdb.com/title/tt2543312/", "The 80s PC boom, as drama"),
			res("doc-silicon-valley", "Silicon Valley", "https://www.imdb.com/title/tt2575988/", "The funniest accurate satire of startups"),
			res("doc-mr-robot", "Mr. Robot", "https://www.imdb.com/title/tt4158110/", "Hacking, told properly"),
			res("doc-computerphile", "Computerphile", "https://www.youtube.com/@Computerphile", "Short, deep videos on how computers work"),
			res("doc-ben-eater", "Ben Eater", "https://www.youtube.com/@BenEater", "Build a computer from scratch on breadboards"),
			res("doc-chm", "Computer History Museum", "https://computerhistory.org", "The real machines, preserved"),
		},
	},

	// ---------- Entertainment ----------
	"anime": {
		Slug:  "anime",
		Name:  "Anime",
		Blurb: "Where to watch, track and talk about it",
		Resources: []Resource{
			res("anime-mal", "MyAnimeList", "https://myanimelist.net", "The database and ranking everyone uses"),
			res("anime-anilist", "AniList", "https://anilist.co", "Cleaner tracking and better stats"),
			res("anime-crunchyroll", "Crunchyroll", "https://www.crunchyroll.com", "The biggest legal streaming library"),
			res("anime-netflix", "Netflix Anime", "https://www.netflix.com/browse/genre/7424", "Originals and exclusives"),
			res("anime-hidive", "HIDIVE", "https://www.hidive.com", "Niche and older titles"),
			res("anime-ann", "Anime News Network", "https://www.animenewsnetwork.com", "News, reviews and an encyclopedia"),
			res("anime-kitsu", "Kitsu", "https://kitsu.app", "Social tracking and lists"),
			res("anime-anichart", "AniChart", "https://anichart.net", "What airs this season"),
			res("anime-mangaplus", "Manga Plus", "https://mangaplus.shueisha.co.jp", "Official free manga chapters"),
			res("anime-corner", "Anime Corner", "https://animecorner.me", "Weekly rankings and polls"),
			res("anime-sakuga", "Sakugabooru", "https://www.sakugabooru.com", "Animation craft, frame by frame"),
			res("anime-anidb", "AniDB", "https://anidb.net", "The most detailed episode data"),
			res("anime-reddit", "r/anime", "https://www.reddit.com/r/anime/", "Discussion and rewatch threads"),
			res("anime-justwatch", "JustWatch", "https://www.justwatch.com", "Find which service has it"),
			res("anime-tvtropes", "TV Tropes: Anime", "https://tvtropes.org", "Every trope, catalogued"),
		},
	},
	"cartoons-movies": {
		Slug:  "cartoons-movies",
		Name:  "Cartoons & Movies",
		Blurb: "Films, animation and where to find them",
		Resources: []Resource{
			res("cm-letterboxd", "Letterboxd", "https://letterboxd.com", "Track, rate and review what you watch"),
			res("cm-imdb", "IMDb", "https://www.imdb.com", "Cast, crew and everything else"),
			res("cm-justwatch", "JustWatch", "https://www.justwatch.com", "Which service has it, in your country"),
			res("cm-rt", "Rotten Tomatoes", "https://www.rottentomatoes.com", "Critic and audience scores"),
			res("cm-disney", "Disney+", "https://www.disneyplus.com", "Disney, Pixar, Marvel, Star Wars"),
			res("cm-netflix", "Netflix", "https://www.netflix.com", "Originals and a huge back catalogue"),
			res("cm-prime", "Prime Video", "https://www.primevideo.com", "Included with Prime, plus rentals"),
			res("cm-tubi", "Tubi", "https://tubitv.com", "Free with ads, surprisingly good"),
			res("cm-pluto", "Pluto TV", "https://pluto.tv", "Free live channels, no account needed"),
			res("cm-archive", "Internet Archive: Animation", "https://archive.org/details/animationandcartoons", "Public domain cartoons, free forever"),
			res("cm-cartoon-network", "Cartoon Network", "https://www.cartoonnetwork.com", "Games and episodes"),
			res("cm-cartoonbrew", "Cartoon Brew", "https://www.cartoonbrew.com", "Animation industry news"),
			res("cm-animationmag", "Animation Magazine", "https://www.animationmagazine.net", "The trade paper of animation"),
			res("cm-kanopy", "Kanopy", "https://www.kanopy.com", "Free with a library card, real cinema"),
			res("cm-tvtropes", "TV Tropes: Film", "https://tvtropes.org/pmwiki/pmwiki.php/Main/Film", "Why that ending felt familiar"),
		},
	},
	"music": {
		Slug:  "music",
		Name:  "Music",
		Blurb: "Listen, learn theory, make your own",
		Resources: []Resource{
			res("music-spotify", "Spotify", "https://open.spotify.com", "Everything, with playlists that work"),
			res("music-ytmusic", "YouTube Music", "https://music.youtube.com", "Live sets and rare recordings"),
			res("music-bandcamp", "Bandcamp", "https://bandcamp.com", "Buy straight from the artist"),
			res("music-soundcloud", "SoundCloud", "https://soundcloud.com", "New and unsigned artists first"),
			res("music-theory", "musictheory.net", "https://www.musictheory.net", "Free lessons from the ground up"),
			res("music-hooktheory", "Hooktheory", "https://www.hooktheory.com", "Why songs sound the way they do"),
			res("music-discogs", "Discogs", "https://www.discogs.com", "Every pressing ever made"),
			res("music-rym", "Rate Your Music", "https://rateyourmusic.com", "The most serious music community"),
			res("music-ableton", "Learning Music (Ableton)", "https://learningmusic.ableton.com", "Make music in your browser"),
			res("music-audacity", "Audacity", "https://www.audacityteam.org", "Free recording and editing"),
			res("music-musescore", "MuseScore", "https://musescore.com", "Free sheet music, millions of scores"),
			res("music-teoria", "Teoria", "https://www.teoria.com", "Exercises for ear and theory"),
			res("music-nts", "NTS Radio", "https://www.nts.live", "Endless radio from real selectors"),
			res("music-fma", "Free Music Archive", "https://freemusicarchive.org", "Free music you can actually use"),
			res("music-whosampled", "WhoSampled", "https://www.whosampled.com", "Where that sample came from"),
		},
	},
	"games": {
		Slug:  "games",
		Name:  "Games",
		Blurb: "Play, follow and learn to build them",
		Resources: []Resource{
			res("games-steam", "Steam", "https://store.steampowered.com", "The biggest store on PC"),
			res("games-itch", "itch.io", "https://itch.io", "Indies, experiments, free games"),
			res("games-epic", "Epic Games Store", "https://store.epicgames.com", "Free game every week"),
			res("games-gog", "GOG", "https://www.gog.com", "No DRM, old games made to run"),
			res("games-hltb", "How Long To Beat", "https://howlongtobeat.com", "Do you have time for it?"),
			res("games-pcgamingwiki", "PCGamingWiki", "https://www.pcgamingwiki.com", "Fix that game before you play"),
			res("games-gmtk", "Game Maker's Toolkit", "https://www.youtube.com/@GMTK", "How games are designed"),
			res("games-noclip", "Noclip", "https://www.youtube.com/@Noclip", "Documentaries about game studios"),
			res("games-backloggd", "Backloggd", "https://backloggd.com", "Letterboxd, but for games"),
			res("games-opencritic", "OpenCritic", "https://opencritic.com", "All the review scores in one place"),
			res("games-chess", "Chess.com", "https://www.chess.com", "Learn and play chess"),
			res("games-lichess", "Lichess", "https://lichess.org", "Free forever, no ads"),
			res("games-gamedev", "GameDev.net", "https://www.gamedev.net", "Forums and articles on making games"),
			res("games-godot", "Godot Engine", "https://godotengine.org", "Free engine to build your own"),
			res("games-proton", "ProtonDB", "https://www.protondb.com", "Will it run on Linux?"),
		},
	},

	// ---------- Learning ----------
	"books": {
		Slug:  "books",
		Name:  "Books & Reading",
		Blurb: "Free libraries, tracking and good recommendations",
		Resources: []Resource{
			res("books-goodreads", "Goodreads", "https://www.goodreads.com", "The biggest reading tracker"),
			res("books-storygraph", "The StoryGraph", "https://app.thestorygraph.com", "Better stats, no Amazon"),
			res("books-gutenberg", "Project Gutenberg", "https://www.gutenberg.org", "75,000 free classic books"),
			res("books-openlibrary", "Open Library", "https://openlibrary.org", "Borrow books from the internet"),
			res("books-standardebooks", "Standard Ebooks", "https://standardebooks.org", "Free classics, beautifully typeset"),
			res("books-librivox", "LibriVox", "https://librivox.org", "Free public domain audiobooks"),
			res("books-archive", "Internet Archive: Texts", "https://archive.org/details/texts", "Millions of scanned books"),
			res("books-worldcat", "WorldCat", "https://search.worldcat.org", "Find it in a real library near you"),
			res("books-bookmarks", "Book Marks", "https://bookmarks.reviews", "Reviews from real critics"),
			res("books-literal", "Literal", "https://literal.club", "A quieter social network for readers"),
			res("books-poetry", "Poetry Foundation", "https://www.poetryfoundation.org", "Poems, free, forever"),
			res("books-opentextbooks", "Open Textbook Library", "https://open.umn.edu/opentextbooks", "Free university textbooks"),
			res("books-wikisource", "Wikisource", "https://en.wikisource.org", "Source texts and translations"),
			res("books-longform", "Longform", "https://longform.org", "The best long articles, curated"),
			res("books-manybooks", "ManyBooks", "https://manybooks.net", "Free ebooks in every format"),
		},
	},
	"science": {
		Slug:  "science",
		Name:  "Science & Space",
		Blurb: "From NASA to the papers themselves",
		Resources: []Resource{
			res("sci-nasa", "NASA", "https://www.nasa.gov", "Missions, images, live data"),
			res("sci-esa", "European Space Agency", "https://www.esa.int", "Europe's missions and science"),
			res("sci-arxiv", "arXiv", "https://arxiv.org", "Preprints, free, the day they are written"),
			res("sci-nature", "Nature", "https://www.nature.com", "The journal everyone cites"),
			res("sci-sciencedaily", "ScienceDaily", "https://www.sciencedaily.com", "New research in plain language"),
			res("sci-quanta", "Quanta Magazine", "https://www.quantamagazine.org", "The best science writing there is"),
			res("sci-khan", "Khan Academy: Science", "https://www.khanacademy.org/science", "Free courses from physics to biology"),
			res("sci-phet", "PhET Simulations", "https://phet.colorado.edu", "Play with physics in the browser"),
			res("sci-ocw", "MIT OpenCourseWare", "https://ocw.mit.edu", "Real MIT lectures, free"),
			res("sci-kurzgesagt", "Kurzgesagt", "https://www.youtube.com/@kurzgesagt", "Big ideas, beautifully animated"),
			res("sci-veritasium", "Veritasium", "https://www.youtube.com/@veritasium", "Science with the experiments included"),
			res("sci-pbs", "PBS Space Time", "https://www.youtube.com/@pbsspacetime", "Astrophysics, properly explained"),
			res("sci-mathworld", "Wolfram MathWorld", "https://mathworld.wolfram.com", "Every definition you need"),
			res("sci-3b1b", "3Blue1Brown", "https://www.3blue1brown.com", "Math you can see"),
			res("sci-pubmed", "PubMed", "https://pubmed.ncbi.nlm.nih.gov", "The medical literature itself"),
		},
	},
	"world-languages": {
		Slug:  "world-languages",
		Name:  "World Languages",
		Blurb: "Learn to actually speak something new",
		Resources: []Resource{
			res("wl-duolingo", "Duolingo", "https://www.duolingo.com", "The habit builder"),
			res("wl-anki", "Anki", "https://apps.ankiweb.net", "Spaced repetition, the real method"),
			res("wl-memrise", "Memrise", "https://www.memrise.com", "Vocabulary with real speakers"),
			res("wl-lingq", "LingQ", "https://www.lingq.com", "Learn by reading things you like"),
			res("wl-tandem", "Tandem", "https://tandem.net", "Talk to native speakers"),
			res("wl-hellotalk", "HelloTalk", "https://www.hellotalk.com", "Language exchange in chat"),
			res("wl-forvo", "Forvo", "https://forvo.com", "Hear any word pronounced by a native"),
			res("wl-wiktionary", "Wiktionary", "https://en.wiktionary.org", "Every word, every language"),
			res("wl-bbc", "BBC Learning English", "https://www.bbc.co.uk/learningenglish", "Free structured English"),
			res("wl-coffee-break", "Coffee Break Languages", "https://coffeebreaklanguages.com", "Podcasts that teach as you walk"),
			res("wl-reactor", "Language Reactor", "https://www.languagereactor.com", "Learn from Netflix subtitles"),
			res("wl-easy-languages", "Easy Languages", "https://www.easy-languages.org", "Street interviews with subtitles"),
			res("wl-italki", "italki", "https://www.italki.com", "Cheap lessons with real teachers"),
			res("wl-reverso", "Reverso Context", "https://context.reverso.net", "See words used in real sentences"),
			res("wl-reddit", "r/languagelearning", "https://www.reddit.com/r/languagelearning/", "What actually works, from learners"),
		},
	},

	// ---------- Lifestyle ----------
	"fitness": {
		Slug:  "fitness",
		Name:  "Fitness & Health",
		Blurb: "Training, food and sleep that hold up",
		Resources: []Resource{
			res("fit-reddit", "r/Fitness Wiki", "https://www.reddit.com/r/Fitness/wiki/index", "The best free training guide online"),
			res("fit-musclewiki", "MuscleWiki", "https://musclewiki.com", "Click a muscle, get the exercise"),
			res("fit-exrx", "ExRx", "https://exrx.net", "The reference for every lift"),
			res("fit-stronglifts", "StrongLifts", "https://stronglifts.com", "Simple strength programme, free"),
			res("fit-ntc", "Nike Training Club", "https://www.nike.com/ntc-app", "Free coached workouts"),
			res("fit-fitnessblender", "Fitness Blender", "https://www.fitnessblender.com", "Full workouts, no fluff"),
			res("fit-darebee", "Darebee", "https://darebee.com", "Free programmes you can do anywhere"),
			res("fit-mfp", "MyFitnessPal", "https://www.myfitnesspal.com", "Count what you actually eat"),
			res("fit-cronometer", "Cronometer", "https://cronometer.com", "Micronutrients, not just calories"),
			res("fit-who", "WHO: Physical Activity", "https://www.who.int/news-room/fact-sheets/detail/physical-activity", "What the guidelines actually say"),
			res("fit-pubmed", "PubMed", "https://pubmed.ncbi.nlm.nih.gov", "Check the claim yourself"),
			res("fit-examine", "Examine", "https://examine.com", "Supplements, evidence-graded"),
			res("fit-yoga", "Yoga With Adriene", "https://www.youtube.com/@yogawithadriene", "Free yoga for every level"),
			res("fit-strava", "Strava", "https://www.strava.com", "Track runs and rides"),
			res("fit-sleep", "Sleep Foundation", "https://www.sleepfoundation.org", "The half of training people skip"),
		},
	},
	"cooking": {
		Slug:  "cooking",
		Name:  "Cooking",
		Blurb: "Technique first, recipes second",
		Resources: []Resource{
			res("cook-seriously", "Serious Eats", "https://www.seriouseats.com", "The why behind the recipe"),
			res("cook-nyt", "NYT Cooking", "https://cooking.nytimes.com", "Reliable recipes, well tested"),
			res("cook-bbc", "BBC Good Food", "https://www.bbcgoodfood.com", "Weeknight basics done right"),
			res("cook-allrecipes", "Allrecipes", "https://www.allrecipes.com", "Whatever is in your fridge"),
			res("cook-kenji", "Kenji Lopez-Alt", "https://www.youtube.com/@JKenjiLopezAlt", "Food science in a home kitchen"),
			res("cook-babish", "Basics with Babish", "https://www.youtube.com/@babishculinaryuniverse", "Learn the technique, then riff"),
			res("cook-chefsteps", "ChefSteps", "https://www.chefsteps.com", "Modernist methods, explained"),
			res("cook-budgetbytes", "Budget Bytes", "https://www.budgetbytes.com", "Good food, cheap"),
			res("cook-kingarthur", "King Arthur Baking", "https://www.kingarthurbaking.com", "Baking that actually works"),
			res("cook-food52", "Food52", "https://food52.com", "Community recipes worth keeping"),
			res("cook-usda", "USDA FoodData Central", "https://fdc.nal.usda.gov", "What is really in your food"),
			res("cook-tasty", "Tasty", "https://tasty.co", "Fast ideas when you are stuck"),
			res("cook-epicurious", "Epicurious", "https://www.epicurious.com", "Classics and technique guides"),
			res("cook-reddit", "r/Cooking", "https://www.reddit.com/r/Cooking/", "Ask why your sauce broke"),
			res("cook-sfah", "Salt Fat Acid Heat", "https://www.saltfatacidheat.com", "The four things that matter"),
		},
	},
	"travel": {
		Slug:  "travel",
		Name:  "Travel",
		Blurb: "Plan it, book it, and know where you are going",
		Resources: []Resource{
			res("travel-flights", "Google Flights", "https://www.google.com/travel/flights", "Fastest way to compare flights"),
			res("travel-skyscanner", "Skyscanner", "https://www.skyscanner.net", "Whole-month price views"),
			res("travel-rome2rio", "Rome2Rio", "https://www.rome2rio.com", "Every way to get from A to B"),
			res("travel-booking", "Booking.com", "https://www.booking.com", "Free cancellation, mostly"),
			res("travel-airbnb", "Airbnb", "https://www.airbnb.com", "Stay where hotels are not"),
			res("travel-hostelworld", "Hostelworld", "https://www.hostelworld.com", "Cheap beds, good company"),
			res("travel-atlas", "Atlas Obscura", "https://www.atlasobscura.com", "The places worth the detour"),
			res("travel-wikivoyage", "Wikivoyage", "https://en.wikivoyage.org", "Free travel guide, written by travellers"),
			res("travel-lonelyplanet", "Lonely Planet", "https://www.lonelyplanet.com", "The classic guidebooks"),
			res("travel-seat61", "The Man in Seat 61", "https://www.seat61.com", "How to travel by train, anywhere"),
			res("travel-nomadlist", "Nomad List", "https://nomadlist.com", "Where to work from, compared"),
			res("travel-state", "US Travel Advisories", "https://travel.state.gov", "Check before you book"),
			res("travel-maps", "Google Maps", "https://maps.google.com", "Offline maps save trips"),
			res("travel-flixbus", "FlixBus", "https://global.flixbus.com", "Cheap coaches across Europe"),
			res("travel-tripadvisor", "TripAdvisor", "https://www.tripadvisor.com", "Reviews, with the usual salt"),
		},
	},

	// ---------- Create & Work ----------
	"design": {
		Slug:  "design",
		Name:  "Design",
		Blurb: "Interfaces, type and colour that work",
		Resources: []Resource{
			res("design-figma", "Figma", "https://www.figma.com", "Design in the browser, free tier"),
			res("design-dribbble", "Dribbble", "https://dribbble.com", "What good looks like"),
			res("design-behance", "Behance", "https://www.behance.net", "Full case studies"),
			res("design-awwwards", "Awwwards", "https://www.awwwards.com", "The best of the web"),
			res("design-mobbin", "Mobbin", "https://mobbin.com", "Real screens from real apps"),
			res("design-refactoringui", "Refactoring UI", "https://www.refactoringui.com", "Why your design looks off"),
			res("design-lawsofux", "Laws of UX", "https://lawsofux.com", "The rules behind good interfaces"),
			res("design-fonts", "Google Fonts", "https://fonts.google.com", "Free fonts that pair well"),
			res("design-coolors", "Coolors", "https://coolors.co", "Build a palette that works"),
			res("design-unsplash", "Unsplash", "https://unsplash.com", "Free photos, no attribution needed"),
			res("design-canva", "Canva", "https://www.canva.com", "Fast, good enough, no software"),
			res("design-material", "Material Design 3", "https://m3.material.io", "Google's system, documented"),
			res("design-nng", "Nielsen Norman Group", "https://www.nngroup.com", "Research on what people actually do"),
			res("design-typescale", "Type Scale", "https://typescale.com", "Get your sizes right"),
			res("design-fontpair", "Fontpair", "https://www.fontpair.co", "Fonts that go together"),
		},
	},
	"business": {
		Slug:  "business",
		Name:  "Business & Money",
		Blurb: "Start something, or just understand money",
		Resources: []Resource{
			res("biz-yc", "Y Combinator Startup School", "https://www.startupschool.org", "Free course on starting a company"),
			res("biz-investopedia", "Investopedia", "https://www.investopedia.com", "Every finance term, explained"),
			res("biz-hbr", "Harvard Business Review", "https://hbr.org", "Management, researched"),
			res("biz-stratechery", "Stratechery", "https://stratechery.com", "The best tech business analysis"),
			res("biz-pg", "Paul Graham: Essays", "https://paulgraham.com/articles.html", "How startups actually work"),
			res("biz-indiehackers", "Indie Hackers", "https://www.indiehackers.com", "People who built small businesses"),
			res("biz-stripe", "Stripe Atlas Guides", "https://stripe.com/atlas/guides", "Practical guides to starting up"),
			res("biz-khan", "Khan Academy: Personal Finance", "https://www.khanacademy.org/college-careers-more/personal-finance", "Money basics, free"),
			res("biz-nerdwallet", "NerdWallet", "https://www.nerdwallet.com", "Compare cards and accounts"),
			res("biz-edgar", "SEC EDGAR", "https://www.sec.gov/edgar/search/", "Read a company's real filings"),
			res("biz-crunchbase", "Crunchbase", "https://www.crunchbase.com", "Who funded whom"),
			res("biz-lean", "Lean Canvas", "https://leanstack.com/lean-canvas", "One page to test an idea"),
			res("biz-bogleheads", "Bogleheads", "https://www.bogleheads.org", "Boring investing that works"),
			res("biz-a16z", "Andreessen Horowitz", "https://a16z.com", "Where the industry is going"),
			res("biz-shopify", "Shopify Learn", "https://www.shopify.com/learn", "Sell something online"),
		},
	},
	"photography": {
		Slug:  "photography",
		Name:  "Photography",
		Blurb: "Take better photos, then edit them",
		Resources: []Resource{
			res("photo-petapixel", "PetaPixel", "https://petapixel.com", "News and technique"),
			res("photo-dpreview", "DPReview", "https://www.dpreview.com", "The camera reviews that matter"),
			res("photo-500px", "500px", "https://500px.com", "Serious work to learn from"),
			res("photo-flickr", "Flickr", "https://www.flickr.com", "EXIF data on every photo"),
			res("photo-unsplash", "Unsplash", "https://unsplash.com", "Free photos and inspiration"),
			res("photo-kenrockwell", "Ken Rockwell", "https://www.kenrockwell.com", "Blunt, useful gear advice"),
			res("photo-cambridge", "Cambridge in Colour", "https://www.cambridgeincolour.com", "The technical side, clearly"),
			res("photo-lightroom", "Lightroom Tutorials", "https://helpx.adobe.com/lightroom-cc/tutorials.html", "Official editing walkthroughs"),
			res("photo-dps", "Digital Photography School", "https://digital-photography-school.com", "Beginner to confident"),
			res("photo-fstoppers", "Fstoppers", "https://fstoppers.com", "Industry and technique"),
			res("photo-photonet", "Photo.net", "https://photo.net", "One of the oldest communities"),
			res("photo-lensculture", "LensCulture", "https://www.lensculture.com", "Contemporary photography"),
			res("photo-magnum", "Magnum Photos", "https://www.magnumphotos.com", "The photographers who defined the medium"),
			res("photo-life", "Photography Life", "https://photographylife.com", "Long, careful explainers"),
			res("photo-joseph", "PhotoJoseph", "https://www.youtube.com/@PhotoJoseph", "Gear and editing, explained"),
		},
	}, "read-manga": {
		Slug:  "read-manga",
		Name:  "Read Manga",
		Blurb: "15 places to read manga, official publishers first",
		Resources: []Resource{
			res("manga-mangaplus", "MANGA Plus by Shueisha", "https://mangaplus.shueisha.co.jp", "Official and free, same-day Shonen Jump chapters"),
			res("manga-viz", "VIZ Media", "https://www.viz.com", "Official English manga and the Shonen Jump app"),
			res("manga-shonenjump", "Shonen Jump", "https://www.viz.com/shonenjump", "Weekly chapters in official translation"),
			res("manga-mangadex", "MangaDex", "https://mangadex.org", "The biggest free library, fan-run"),
			res("manga-crunchyroll", "Crunchyroll Manga", "https://www.crunchyroll.com/comics/manga", "Manga included with a Crunchyroll account"),
			res("manga-kodansha", "Kodansha USA", "https://kodansha.us", "Official publisher with free first chapters"),
			res("manga-yenpress", "Yen Press", "https://yenpress.com", "Official publisher, read online"),
			res("manga-sevenseas", "Seven Seas Entertainment", "https://sevenseasentertainment.com", "Official publisher with generous free samples"),
			res("manga-darkhorse", "Dark Horse Manga", "https://www.darkhorse.com", "Official publisher for sci-fi and horror"),
			res("manga-azuki", "Azuki", "https://www.azuki.co", "Subscription reader with a curated shelf"),
			res("manga-comikey", "Comikey", "https://comikey.com", "Officially licensed, many titles free"),
			res("manga-inkr", "INKR Comics", "https://inkr.com", "Officially licensed manga and manhwa"),
			res("manga-mangaplanet", "Manga Planet", "https://mangaplanet.com", "Official subscription service"),
			res("manga-bookwalker", "BOOK\u2606WALKER", "https://global.bookwalker.jp", "Buy official digital manga"),
			res("manga-mangaupdates", "MangaUpdates", "https://www.mangaupdates.com", "Track releases and find what to read next"),
		},
	},
	"read-manhwa": {
		Slug:  "read-manhwa",
		Name:  "Read Manhwa",
		Blurb: "15 places to read Korean manhwa, official platforms first",
		Resources: []Resource{
			res("manhwa-webtoon", "WEBTOON", "https://www.webtoons.com", "The home of Korean webtoons, free daily chapters"),
			res("manhwa-tappytoon", "Tappytoon", "https://www.tappytoon.com", "Official manhwa, romance and action"),
			res("manhwa-tapas", "Tapas", "https://tapas.io", "Official and indie comics, plenty free"),
			res("manhwa-lezhin", "Lezhin Comics", "https://www.lezhin.com/en", "Official manhwa, mature titles"),
			res("manhwa-manta", "Manta", "https://manta.net", "Unlimited official manhwa subscription"),
			res("manhwa-toomics", "Toomics", "https://toomics.com", "Official manhwa across many genres"),
			res("manhwa-pocketcomics", "Pocket Comics", "https://www.pocketcomics.com", "Official manhwa with free chapters daily"),
			res("manhwa-copin", "Copin Comics", "https://copincomics.com", "Official manhwa publisher"),
			res("manhwa-inkr", "INKR Comics", "https://inkr.com", "Officially licensed manhwa"),
			res("manhwa-kakaopage", "Kakao Page", "https://page.kakao.com", "The Korean original, where most series start"),
			res("manhwa-naver", "Naver Series", "https://series.naver.com", "Korea's biggest webtoon platform"),
			res("manhwa-canvas", "WEBTOON Canvas", "https://www.webtoons.com/en/canvas", "Free indie webtoons"),
			res("manhwa-mangadex", "MangaDex", "https://mangadex.org", "Big free library, manhwa included"),
			res("manhwa-bato", "Bato.to", "https://bato.to", "Community library with a manhwa section"),
			res("manhwa-mangaupdates", "MangaUpdates", "https://www.mangaupdates.com", "Track what you are reading"),
		},
	},
	"read-comix": {
		Slug:  "read-comix",
		Name:  "Read Comix",
		Blurb: "15 places to read comics, official publishers and free libraries first",
		Resources: []Resource{
			res("comix-marvel", "Marvel Unlimited", "https://www.marvel.com/unlimited", "Every Marvel comic, one subscription"),
			res("comix-dc", "DC Universe Infinite", "https://www.dcuniverseinfinite.com", "DC's own library, thousands of issues"),
			res("comix-globalcomix", "GlobalComix", "https://globalcomix.com", "Huge indie and publisher catalogue, free to start"),
			res("comix-hoopla", "Hoopla", "https://www.hoopladigital.com", "Free comics with a library card"),
			res("comix-libby", "Libby", "https://libbyapp.com", "Borrow graphic novels free from your library"),
			res("comix-image", "Image Comics", "https://imagecomics.com", "The Walking Dead, Saga, and free first issues"),
			res("comix-darkhorse", "Dark Horse Digital", "https://digital.darkhorse.com", "Official store for Hellboy and sci-fi"),
			res("comix-boom", "BOOM! Studios", "https://www.boom-studios.com", "Official publisher, strong original series"),
			res("comix-idw", "IDW Publishing", "https://www.idwpublishing.com", "Official publisher for licensed and indie titles"),
			res("comix-archie", "Archie Comics", "https://www.archiecomics.com", "The classics, plus the modern reboots"),
			res("comix-gocomics", "GoComics", "https://www.gocomics.com", "Daily newspaper strips, free"),
			res("comix-xkcd", "xkcd", "https://xkcd.com", "Science, sarcasm and stick figures"),
			res("comix-ksbd", "Kill Six Billion Demons", "https://killsixbilliondemons.com", "A free webcomic with astonishing art"),
			res("comix-girlgenius", "Girl Genius", "https://www.girlgeniusonline.com", "Long-running free steampunk adventure"),
			res("comix-league", "League of Comic Geeks", "https://leagueofcomicgeeks.com", "Track what you read and what drops next"),
		},
	},
}

// sections is the whole site in order. Add a topic here and it appears
// everywhere: the section page, the browse page, and the redirect handler.
var sections = []Section{
	{
		Slug:    "programming",
		Name:    "Programming",
		Tagline: "Code",
		Blurb:   "Learn a language, or watch the story of the people who built them first.",
		Theme:   "prog",
		Forks: []Fork{
			{Title: "Watch documentaries", Blurb: "Pick a language, then see where to watch the films and talks about it.", Slug: "documentaries", Href: "/section/programming/documentaries"},
			{Title: "Straight to learning", Blurb: "Pick a language, then get its fifteen places: courses, practice grounds and project ideas.", Slug: "learning", Href: "/section/programming/learning"},
			{Title: "Docs", Blurb: "Pick a language, then read its documentation and references.", Slug: "docs", Href: "/section/programming/docs"},
		},
		Topics: programmingTopicList(),
	},
	{
		Slug:    "entertainment",
		Name:    "Entertainment",
		Tagline: "Anime · Film · Music",
		Blurb:   "The good stuff, without the twenty open tabs.",
		Theme:   "ent",
		Topics: []Topic{
			otherTopics["anime"],
			otherTopics["cartoons-movies"],
			otherTopics["music"],
			otherTopics["games"],
		},
	},
	{
		Slug:    "learning",
		Name:    "Learning",
		Tagline: "Books · Science",
		Blurb:   "Reading, research and languages, from free libraries to the papers themselves.",
		Theme:   "learn",
		Topics: []Topic{
			otherTopics["books"],
			otherTopics["science"],
			otherTopics["world-languages"],
		},
	},
	{
		Slug:    "lifestyle",
		Name:    "Lifestyle",
		Tagline: "Fitness · Food · Travel",
		Blurb:   "Training, cooking and going places, with sources that hold up.",
		Theme:   "life",
		Topics: []Topic{
			otherTopics["fitness"],
			otherTopics["cooking"],
			otherTopics["travel"],
		},
	},
	{
		Slug:    "create-work",
		Name:    "Create & Work",
		Tagline: "Design · Money",
		Blurb:   "Make things people look at, and understand the money around them.",
		Theme:   "work",
		Topics: []Topic{
			otherTopics["design"],
			otherTopics["business"],
			otherTopics["photography"],
		},
	},
	{
		Slug:    "manga",
		Name:    "Read Manga",
		Tagline: "Manga",
		Blurb:   "Fifteen places to actually read manga, official publishers first.",
		Theme:   "manga",
		Topics:  []Topic{otherTopics["read-manga"]},
		Direct:  true,
	},
	{
		Slug:    "manhwa",
		Name:    "Read Manhwa",
		Tagline: "Manhwa",
		Blurb:   "Fifteen places to read Korean manhwa, official platforms first.",
		Theme:   "manhwa",
		Topics:  []Topic{otherTopics["read-manhwa"]},
		Direct:  true,
	},
	{
		Slug:    "comix",
		Name:    "Read Comix",
		Tagline: "Comics",
		Blurb:   "Fifteen places to read comics, official publishers and free libraries first.",
		Theme:   "comix",
		Topics:  []Topic{otherTopics["read-comix"]},
		Direct:  true,
	},
}

// programmingTopicList returns the programming languages in a stable order.
func programmingTopicList() []Topic {
	list := make([]Topic, 0, len(programmingTopics))
	for _, t := range programmingTopics {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// Lookup tables, built once at start-up.
var (
	topicBySlug    = map[string]Topic{}
	sectionBySlug  = map[string]Section{}
	sectionOfTopic = map[string]string{}
)

func init() {
	for _, f := range sections {
		sectionBySlug[f.Slug] = f
		for _, t := range f.Topics {
			topicBySlug[t.Slug] = t
			sectionOfTopic[t.Slug] = f.Slug
		}
	}
	// The documentary fork is reachable but not listed as a box.
	for _, f := range sections {
		for _, fk := range f.Forks {
			if fk.Slug == "" {
				continue
			}
			if t, ok := otherTopics[fk.Slug]; ok {
				topicBySlug[t.Slug] = t
				sectionOfTopic[t.Slug] = f.Slug
			}
		}
	}
}

// themeFor returns the css theme key for a page.
func themeFor(slug string) string {
	if f, ok := sectionBySlug[slug]; ok {
		return f.Theme
	}
	return "base"
}
