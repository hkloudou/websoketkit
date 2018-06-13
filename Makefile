git:
	-git add .
	-git commit -m 'build auto commit'
	-git tag -f 0.1.2
	-git push origin master -f --tags
