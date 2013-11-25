#!/usr/bin/env bash

# A set of utility functions which integrate
# rd with bash.
#
# Add 'source <path>/rd.bash' to ~/.bashrc
# to make these facilities available in new shells.

# cdr - Switch to a recently used directory.
#
# Usage:
#   cdr <pattern>
#   cdr <id>
#
# The first form searches for a recently used dir matching a given
# pattern. If there is a single result, cdr switches
# to the given dir. If there are multiple matches,
# prints a list of matches alongside a numeric ID for
# each.
#
# The second form 'cdr <id>' can then be used to switch
# to a dir listed by a recent 'cdr <pattern>' query.
#
function cdr {
	matches=`rd $@`

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
