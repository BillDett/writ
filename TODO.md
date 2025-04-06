# TODO in writ

Application
* BUG: We lose all focus on the 3 widgets sometimes- need a way to ensure one widget always has focus
* BUG: CTRL-E will go into Editor even if no document is selected- should disallow that
* Startup flow is a little clunky- is there a better way to 'initialize' the UI before it's usable while
  still being able to use it (e.g. pop up error messages, etc). Need a way to 'init' the app after the 
  application Run() has started.
* Make the Organizer come and go by adding/removing from the Grid with a hotkey (full screen editing)
* Need an autobackup goroutine that periodically ships the entire database to cloud storage when on wifi.


Organizer
  * Filtered (CTRL-F) (using a search expression via the inputField )
    * This means we need to explore how to enable the FTS5 library in our sqlite instance and 
    refactor how we store text.

TextWidget
* We ought to get system-level copy/paste working- tview can help here I think


