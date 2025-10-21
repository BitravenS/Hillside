# Hillside
An end-to-end encrypted p2p cli chat app

## Roadmap

### MVP
The most important features to be ready for release:

- [x] History Syncing
- [x] Key and history resyncing topic (catchup topic)
- [x] Fix "Topic exists" when joining different rooms
- [ ] Encrypt DB
- [ ] Minimize duplicate information in messages
- [ ] Reduce the signature size (2.4 Kb is waaay to big for each message)
- [ ] Basic profile management
- [ ] Sorting servers and rooms by most recently visited
- [ ] Save favourite servers and rooms
- [ ] Close open connections and pubsubs gracefully & shutdown client gracefully
- [ ] Add basic room info to be displayed
- [ ] Organize the packages and clean up the codebase

### Top Prio
Somewhat necessary features to be rolled out gradually after release:

- [ ] Add in chat vim controls
- [ ] Add DMs
- [ ] Add commands (to tag, reply, forward...)
- [ ] Add a different hub config with centralized db (in parallel to the decentralized one, for convenience)
- [ ] Add better security to Private servers and rooms (with SSS perhaps)
- [ ] Add the possibility for rekeying (i.e. change the room key)
- [ ] Add trustworthiness (i.e. someone sent a badly signed message or tried joining with a wrong key)


### Lower Prio & QoL
Lower priority, but still nice to have features to be rolled out eventually:


- [ ] Ability to send images & format messages in markdown
- [ ] Emotes?
- [ ] Democratic server management (vote based or smth)
- [ ] Voice chat (thinking about it gives me a headache)
