[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]
[![LinkedIn][linkedin-shield]][linkedin-url]

<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://github.com/tywin1104/minecraft-gatekeeper">
    <img src="images/gatekeeper.png" alt="Logo" >
  </a>

  <h2 align="center">Minecraft GateKeeper</h1>

  <h4 align="center">
    An all-in-one reliable whitelist application system for your Minecraft server
    <br />
    ·
    <a href="https://github.com/tywin1104/mc-gatekeeper/issues">Request Feature</a>
  </h3>
</p>

<!-- TABLE OF CONTENTS -->

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Introducation](#introducation)
- [Workflows With Screenshots:](#workflows-with-screenshots)
    - [For Players:](#for-players)
    - [For Ops:](#for-ops)
    - [For Server Owner:](#for-server-owner)
- [Deployment & Configurations](#deployment--configurations)
    - [Deployment using docker-compse](#deployment-using-docker-compse)
- [Local Dev Setup](#local-dev-setup)
- [Custimizations](#custimizations)
- [Feature Requests](#feature-requests)
- [License](#license)
- [Contact](#contact)
- [Built With](#built-with)

<!-- ABOUT THE PROJECT -->

## Introducation

Gatekeeper aims to make whitelist application and management for Minecraft server easier than ever before.

What's provided out of the box:

- An easy-to-use self-serve web interface for players to apply for the whitelist
- Email dispatching mechanism for Ops to process incoming requests
- Real-time confirmation and notification emails to users for application status update
- One-click processing workflow for Ops to deny/approve user applications
- Automatic grant whitelist for approved users on the server via RCON
- Real-time management and monitoring dashboard for the server owner
- Rich configuration and customization options (see below)
- Multi-language support

On top of these, it is implemented with security(encryotion; private endpoints token-based authentication) and fault-torlance(retry operations; message queue connection recovery mechaism) in mind.

## Workflows With Screenshots:

#### For Players:

- Players apply for whitelist by filling in required information and complete Minecraft account verification process on the website
  ![Image of Yaktocat](images/Splash.png)
- Once an application is submitted, the player will receive a confirmation email with a link to view his/her current application status
  ![Image of Yaktocat](images/Confirmation.png)
  ![Image of Yaktocat](images/Status.png)
- Once the application is approved, the player's Minecraft username will be granted whitelist on the Minecraft server and the player will receive a notification email
  ![Image of Yaktocat](images/Approve.png)
- If the application is denied, the player will be notified via email as well. The player will be able to re-submit another application

- Key features:
  - Form submission is protected by Google Recaptcha
  - Process in place to verify the player's Minecraft username to prevent identity misuse
  - Disallow repeat/duplicate application from the same user
  - Asynchronous message processing model which improves responsiveness
  - Encoded and encrypted status check the link to prevent access from unintended users

#### For Ops:

- Assigned Ops of each application will be notified via email and will be provided with an action link. On the action page, Ops could view the application details and deny/approve the corresponding application by a single click
  ![Image of Yaktocat](images/ActionEmail.png)
- Optionally, Ops could add a note for that application when making decisions for future references
  ![Image of Yaktocat](images/Action.png)

- Key features:
  - No login required for Ops to simplify the workflow
  - Encryption and encoding in place to prevent misuse: the action page is only visible for Ops who are assigned with a particular application. Once an application is resolved, it will no longer be available.

#### For Server Owner:

- Server owner could view/monitor/manage all applications through the management dashboard which are only accessible with configured root credentials
  ![Image of Yaktocat](images/Dashboard1.png)
  ![Image of Yaktocat](images/Dashboard2.png)

- Key features:
  - Centralized dashboard to manage all applications, view aggregate stats.
  - Able to export application entries to external CSV files
  - The dashboard login page is also protected by Recaptcha to enhance security

## Deployment & Configurations

You can follow the below steps to get the application up and running easily:

#### Deployment using docker-compse
 - Clone the repo onto your workstation / Linux virtual machine(EC2, droplet...)
 - Make sure `docker` and `docker-compose` is installed on your workstation
 - Create `config.yaml` inside the `server` directory. Fill in the values according to your setup. (see config_sample.yaml for reference)

   `mongodbConn: mongodb://mongo:27017`

   `rabbitMQConn: amqp://guest:guest@rabbitmq`

   `redisConn: cache:6379`

   Keep these three configuration value as above.
 - Inside `docker-compose.yaml`, change `RECAPTCHA_SITEKEY` to your recaptcha sitekey(v2 invisible)
 - If you are running in a machine with domain name DNS configured, you need to change `FRONTEND_DEPLOYED_URL` in `docker-compose.yaml` to be your domain address instead of localhost.
 - run `docker-compose up -d`
 - Once the process is finished, go to `http://localhost` or your configured domain address to view the application


## Local Dev Setup

More to come...

[Open API Spec for the backend server](https://app.swaggerhub.com/apis/tywinzhang/mc-whitelist/)

## Custimizations

Here are some possible trivial custimizations that could be done that fits your need. Different from changing the configuration values, these will incur minor code change and require rebuilding the image to your own container registry in order to deploy

- Add/ Change translations for multilanguages

  See `app/locale` directory for translation files

- Add fields to the whitelist request form
- Style custimization for the frontend app page.
- Change Email templates

  See `server/mailer/templates`

- .....
  <!-- LICENSE -->

## Feature Requests

Please open a issue to request a feature or make a contact directly

## License

Distributed under the MIT License. See `LICENSE` for more information

<!-- CONTACT -->

## Contact

Tianyi Zhang - tiaven1104@gmail.com

## Built With

- [React](https://reactjs.org/)
- [Bootstrap](https://getbootstrap.com/)
- [Material UI](https://material-ui.com/)
- [Recaptcha](https://www.google.com/recaptcha/intro/v3.html)
- [Go](https://golang.org/)
- [RabbitMQ](https://www.rabbitmq.com/)
- [MongoDB](https://www.mongodb.com/)

<!-- MARKDOWN LINKS & IMAGES -->

[stars-shield]: https://img.shields.io/github/stars/tywin1104/mc-gatekeeper.svg?style=flat-square
[stars-url]: https://github.com/tywin1104/mc-gatekeeper/stargazers
[issues-shield]: https://img.shields.io/github/issues/tywin1104/mc-gatekeeper.svg?style=flat-square
[issues-url]: https://github.com/tywin1104/mc-gatekeeper/issues
[license-shield]: https://img.shields.io/github/license/tywin1104/mc-gatekeeper.svg?style=flat-square
[license-url]: https://github.com/tywin1104/mc-gatekeeper/blob/master/LICENSE
[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=flat-square&logo=linkedin&colorB=555
[linkedin-url]: https://www.linkedin.com/in/tywinzhang/
