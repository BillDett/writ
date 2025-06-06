# TODO in writ

Application
* BUG: We lose all focus on the 3 widgets sometimes- need a way to ensure one widget always has focus
* BUG: CTRL-E will go into Editor even if no document is selected- should disallow that
* BUG: When starting with an empty database, there is no way to clear the Error modal telling you to create a new document
* Make the Organizer come and go by adding/removing from the Grid with a hotkey (full screen editing)
* Need an autobackup goroutine that periodically ships the entire database to cloud storage when on wifi.


Organizer
  * Filtered (CTRL-F) (using a search expression via the inputField )
    * This means we need to explore how to enable the FTS5 library in our sqlite instance and 
    refactor how we store text.

TextWidget
* 


