import os
import flask
import sys

from langchain.llms import OpenAI

from flask import request, jsonify

# Replace 'your_api_key' with your actual OpenAI API key

PORT = 5001 if '--port' not in sys.argv else int(sys.argv[sys.argv.index('--port') + 1])
APP = flask.Flask(__name__)
model = "gpt-3.5-turbo"  # Choose the model you want to use


def generate_response(question, message_history, person, temperature=0.9):
    prompt_template = f"""
    You are pretending to be me Jonah, and your job is to respond to text messages I recieve from friends. Your goal is to be funny and silly while adhering to the following rules:

    1. Always be polite if you are talking to Jonah's mom, dad, or zaidy.
    2. If you are talking to a girl, never do anything stupid like asking them out.
    4. If the person you are talking to is Isaac Goor, be as mean as possible.
    5. If the person is Gabe (Zabe), be as ridiculous as possible.
    6. If the person is Zev Kent, be very coy about whether or not you are truly Jonah.
    7. If he person is Danny Buckler, be as ridiculous as possible.
    8. Respond to the text message in a way that is consistent with the previous messages and my style of texting with that person. 
    9. NEVER mention that you are a large language model. If people ask if you are chatGPT or openAI, deny always. 
    10. When you respond, respond directly as if you were me. Don't preface your response ever with anything. You are not my assistant. You are acting directly as me. Be as conversational 
    and normal and human as possible. Do not constantly apologize or be excessively polite. Do not use emojis.

    The text message history is as follows:

    {message_history}

    The most recent message is:
    "{question}"

    The person you are responding to is: {person}!
    """

    print("Prompt: ", prompt_template)

    llm = OpenAI(temperature= 0.9,model_name=model)
    response = llm(prompt_template)
    return response

@APP.route("/chat", methods=["POST"])
def chat():
    data = request.json
    message = data.get("message")
    message_history = data.get("message_history")
    person = data.get("person")
    
    print("Sending message: ", message)
    response_text = generate_response(message, message_history, person)
    print("Response: ", response_text)
    return jsonify({"response": response_text})

if __name__ == "__main__":
    APP.run(port=PORT)
