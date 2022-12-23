# joplin2obsidian

## What is joplin2obsidian?
joplin2obsidian is a conversion tool to help you making the move from [Joplin](https://joplinapp.org/) to [Obsidian](https://obsidian.md).

## How to use
1. Download the [latest release](https://github.com/luxi78/joplin2obsidian/releases/latest)

2. Open Joplin, export all data (*notes and corresponding resources*) as RAW to a directory

![export](exportnotes.png)

3. Run `joplin2obsidian` to convert the "RAW - Joplin Export Directory" to an Obsidian-compatible vault directory 
~~~bash
Usage of joplin2obsidian:
  -s string
        Specify the source directory where Joplin exported the RAW data
  -d string
        The destination directory of Obsidian vault
~~~

4. Open the destination directory as vault in Obsidian

Done!

## Build from source
~~~bash
$ git clone https://github.com/luxi78/joplin2obsidian.git
$ cd  joplin2obsidian
$ make
$ cd dist
~~~
