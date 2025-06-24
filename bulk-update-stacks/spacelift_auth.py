import requests


def get_spacelift_auth_token():

    url = "https://<your-spacelift-org>.app.spacelift.io/graphql"
    API_KEY_ID = ''
    API_KEY_SECRET = ''

    get_jwt = """
    mutation GetSpaceliftToken($keyId: ID!, $keySecret: String!) {
        apiKeyUser(id: $keyId, secret: $keySecret) {
            id
            jwt
        }
    }
    """

    variables = {
        "keyId": API_KEY_ID,
        "keySecret": API_KEY_SECRET
    }

    req = requests.post(
        url,
        json={"query": get_jwt, "variables": variables},
    )

    if req.status_code == 200:
        response = req.json()
        API_TOKEN = response['data']['apiKeyUser']['jwt']
    return API_TOKEN
