function cdr {
	matches=`rd $1`

	# Check whether the response was a single match
	# or a list of possible matches
	newLineIndex=`expr index "$matches" ":"`
	if [ $newLineIndex -eq 0 ]
	then
		cd "$matches"
	else
		echo "$matches"
	fi
}
