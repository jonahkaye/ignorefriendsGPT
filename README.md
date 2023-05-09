# ignorefriendsGPT
* Two terminals: `go run main.go`, and `python server.py`

The goal of this is to ignore my friends and mess with them. If they sorta think they are talking to a bot but aren't sure, then I think that is really funny. 

I modified it to:
- use the langchain openAI API
- to know who I am talking
- to have recent messages as history. 
- wait on messages if they are coming in continuously using a channel

* TODO:
* persistent chat history loading. 
* calendar access
* access to other things in my life

This is a fork of Daniel gross's whatsapp-gpt