## Propose
The propose of the app is to wait for all users to sign that they are ready, then show an image simultaneously on all the users phones.

## Context
1. I need it to be an Flutter app, with simple server.
2. I already have an GCP VM where I can upload podman containers or use in anyway to implement the server.
3. The app isn't commercial, don't need to be visually appealing or to have any unnecessary complexity, it's for me and my friends use only.
4. Max of 10 users.
5. Must be simple enough to develop and delivery in a few hours, with LLM help.
6. It don't need anytime of security, authentication or authorization as it will be used just by trusted friends.
7. Probably irrelevant, but the image will be an QR Code.

## Usage flow
1. I need to able to preset the app loading an image to it.
2. Then, all the users need to toggle an button asserting that they are ready.
3. When the last user toggle the button, the app must act synchronously on everyones phone doing the following: vibrate the phone and then show an loading screen for 3 seconds.
4. Finally, the app must show the pre-loaded image simultaneously on everyones phone.

## Usage context
1. Consider that during the usage flow all the users will have the app continuously opened.
2. The app should be resilient to connection instability to assure simultaneity.
3. After the last user toggle the button, as described on Item 3 of Usage Flow, the app may perform extra background steps to ensure simultaneity on starting the actions described. As a way to accomplish last item.
4. No additional users will join the group after all buttons toggled.
