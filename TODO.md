# TODO in writ

Application
* BUG: We lose all focus on the 3 widgets sometimes- need a way to ensure one widget always has focus
* Startup flow is a little clunky- is there a better way to 'initialize' the UI before it's usable while
  still being able to use it (e.g. pop up error messages, etc). Need a way to 'init' the app after the 
  application Run() has started.
* Use a big modal with static text for help (key off F1 or '?' in MainWindow)


Organizer
* Add modality- between Document and Trash mode (CTRL-T toggles) - change title text
* In Trash mode documents can be:
  * Restored (CTRL-Z)
  * Fully Deleted (DEL)
* In Document mode documents can be:
  * Duplicated (CTRL-D)
  * Renamed (CTRL-R)
  * Filtered (CTRL-F) (using a FooterWidget input)

TextWidget
* 

