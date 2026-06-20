def signup(email, full_name):
    db.save({"email": email, "name": full_name})
    requests.post("https://api.stripe.com/v1/customers", json={"email": email})
    log.info("created user %s", email)
