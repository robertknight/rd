#!/usr/bin/env fish

# A set of utility functions which integrate
# rd with the fish shell (fishshell.com)
#
# See rd.bash for documentation
#
function cdr
	set matches (eval "$RD_BIN_PATH" -color query $argv)
	if test (count $matches) -eq 0
		return
	end

	set newLineIndex (expr "$matches" : '.*:.*')
	if test $newLineIndex -eq 0
		cd "$matches"
	else
		for match in $matches
			echo $match
		end
		read -p 'echo -e -n "\nSelect match: "' matchIndex
		if test $matchIndex
			cdr $matchIndex
		end
	end
end
