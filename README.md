## How to use
- Download the release build from the releases tab
- Put names that you want sorted in ```titles.txt``` file next to the executable (each name in the file should be on a separate line)
- Start the program and follow instructions on the screen
- When the program finishes the results will be stored in csv format  

### Pro Tips: 
- letters in parentheses on buttons are shortcuts
- you can close the program any time without loosing intermediate results as long as you don't touch the ```TierMaker.tmp```  file
- if you made a mistake you can go to ```TierMaker.tmp``` (it's just a .txt file) and remove any number of titles from the ```SortedNames``` list, if you do that don't forget to set ```Mid``` property to ```-1``` and make sure you don't leave any extra commas
- if you accidentally mess up ```TierMaker.tmp``` file it will be renamed to ```invalid_tmp_file.tmp``` so you can fix it and rename it back, if you don't fix it, it will be overwritten the next time the program sees an invalid tmp file