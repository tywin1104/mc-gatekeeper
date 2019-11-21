import React from "react";
import {
  Button,
  Form,
  FormGroup,
  Label,
  Input,
  UncontrolledAlert,
  Jumbotron,
  Spinner,
  CardBody,
  Card,
  Badge
} from "reactstrap";
import "./Application.css";
import Container from "@material-ui/core/Container";
import Recaptcha from "react-google-invisible-recaptcha";
import RequestsService from "../service/RequestsService";
import RecaptchaService from "../service/RecaptchaService";
import MinecraftService from "../service/MinecraftService";
import i18next from "i18next";
import QRCode from "qrcode.react";

const RECAPTCHA_SITEKEY = window.RECAPTCHA_SITEKEY
  ? window.RECAPTCHA_SITEKEY
  : process.env.REACT_APP_RECAPTCHA_SITEKEY;
console.log(RECAPTCHA_SITEKEY);

class Application extends React.Component {
  constructor(props) {
    super(props);
    this.onResolved = this.onResolved.bind(this);
    this.state = {
      email: "",
      username: "",
      gender: "male",
      age: "",
      applicationText: "",
      errorMsg: "",
      success: false,
      isOpen: false,
      verified: false,
      loading: false
    };
    this.VERIFICATION_QRCODE_CONTENT = "verified";

    // Username verification related error messages
    this.ERR_SKIN_NOT_FOUND = i18next.t("Splash.SkinNotFoundErrMsg");
    this.ERR_SKIN_NOT_MATCH = i18next.t("Splash.SkinNotMatchErrMsg");
    this.ERR_VERIFICATION = i18next.t("Splash.VerificationErrMsg");
    this.ERR_RATE_LIMIT = i18next.t("Splash.RateLimitErrMsg");
    this.ERR_INVALID_USERNAME = i18next.t("Splash.InvalidUsernameErrMsg");
    this.ERR_EMPTY_USERNAME = i18next.t("Splash.EmptyUsernameErrMsg");
    this.ERR_REPEAT_REQUEST = i18next.t("Splash.RepeatRequestErrMsg");
    this.ERR_ALREADY_APPROVED = i18next.t(
      "Splash.RequestAlreadyApprovedErrMsg"
    );

    // Request submission related error messages
    this.ERR_INTERNAL = i18next.t("Splash.SubmissionInternalErrMsg");
  }

  onToggle = () => {
    this.setState({
      isOpen: !this.state.isOpen
    });
  };

  onResolved() {
    RecaptchaService.verify(this.recaptcha.getResponse()).then(res => {
      if (res.status === 200 && res.data.success) {
        RequestsService.createRequest({
          email: this.state.email,
          username: this.state.username,
          gender: this.state.gender,
          age: parseInt(this.state.age),
          info: {
            applicationText: this.state.applicationText
          }
        })
          .then(res => {
            // Created
            if (res.status === 201) {
              this.setState({
                success: true
              });
            }
          })
          .catch(error => {
            if (error.response) {
              // 422 Unprocessable Entity means there is pending request with that username in the system
              // (duplicate request)
              let statusCode = error.response.status;
              if (statusCode === 422) {
                this.setState({
                  errorMsg: this.ERR_REPEAT_REQUEST
                });
              } else if (statusCode === 409) {
                // 409 Conflict indicates the request with this username is already approved
                this.setState({
                  errorMsg: this.ERR_ALREADY_APPROVED
                });
              } else if (statusCode === 500) {
                this.setState({
                  errorMsg: this.ERR_INTERNAL
                });
              }
            }
          });
      }
    });
  }

  onSubmit = event => {
    event.preventDefault();
    this.recaptcha.reset();
    this.recaptcha.execute();
  };

  handleInputChange = event => {
    const { value, name } = event.target;
    this.setState({
      [name]: value
    });
  };
  downloadQR = () => {
    const canvas = document.getElementById("qrcode");
    const pngUrl = canvas
      .toDataURL("image/png")
      .replace("image/png", "image/octet-stream");
    let downloadLink = document.createElement("a");
    downloadLink.href = pngUrl;
    downloadLink.download = "verify.png";
    document.body.appendChild(downloadLink);
    downloadLink.click();
    document.body.removeChild(downloadLink);
  };

  getUsernameVerificationBadge = () => {
    if (this.state.verified) {
      return (
        <Badge color="success" pill>
          {i18next.t("Splash.Verified")}
        </Badge>
      );
    } else {
      return (
        <Badge color="danger" pill>
          {i18next.t("Splash.NotVerified")}
        </Badge>
      );
    }
  };

  onVerifyUsername = event => {
    event.preventDefault();
    // Start displaying the spinner and temp disable the verify button
    this.setState({ loading: true });
    let username = this.state.username;
    if (!username.trim()) {
      this.setState({
        errorMsg: this.ERR_EMPTY_USERNAME
      });
      return;
    }
    MinecraftService.getSkinImage(username)
      .then(res => {
        if (res.status === 200) {
          let skinImageURL = res.data.skin.url;
          // Double check againest empty url link even if status OK
          // Unexpected case
          if (!skinImageURL) {
            this.setState({
              errorMsg: this.ERR_SKIN_NOT_FOUND
            });
            return;
          }
          MinecraftService.getQRCodeContent(skinImageURL)
            .then(res => {
              // Thie bloack guarantees the skin image url of some sort
              if (!res.data[0].symbol[0].error) {
                // Here indicates the skin image is indeed a valid readable qrcode
                if (
                  res.data[0].symbol[0].data ===
                  this.VERIFICATION_QRCODE_CONTENT
                ) {
                  this.setState({ verified: true });
                } else {
                  this.setState({
                    errorMsg: this.ERR_SKIN_NOT_MATCH
                  });
                }
              } else {
                // Skin image is not a valid readble qrcode
                this.setState({
                  errorMsg: this.ERR_SKIN_NOT_MATCH
                });
              }
              this.setState({ loading: false });
            })
            .catch(error => {
              // Invalid qrcode reader api call
              // Unexpected case
              this.setState({
                errorMsg: this.ERR_VERIFICATION
              });
              this.setState({ loading: false });
            });
        }
      })
      .catch(error => {
        if (error.response) {
          // Rate limit for Mojang API reached
          if (error.response.status === 429) {
            this.setState({
              errorMsg: this.ERR_RATE_LIMIT
            });
          } else if (error.response.status === 400) {
            // getSkinImage will return 400 if unable to get uuid from the given username
            this.setState({
              errorMsg: this.ERR_INVALID_USERNAME
            });
          } else {
            // Internal error caused from unexpected behaviors from Mojang API server
            this.setState({
              errorMsg: this.ERR_VERIFICATION
            });
          }
        }
        this.setState({ loading: false });
      });
  };
  render() {
    let messageBlock;
    if (this.state.errorMsg !== "") {
      messageBlock = (
        <UncontrolledAlert color="danger" fade={false}>
          <span className="alert-inner--icon">
            <i className="ni ni-like-2" />
          </span>{" "}
          <span className="alert-inner--text">
            <strong>Error</strong> {this.state.errorMsg}
          </span>
        </UncontrolledAlert>
      );
      // Error message will disapper after a delay
      setTimeout(() => {
        this.setState({
          errorMsg: ""
        });
      }, 5000);
    } else if (this.state.success) {
      messageBlock = (
        <UncontrolledAlert color="success" fade={false}>
          <span className="alert-inner--icon">
            <i className="ni ni-like-2" />
          </span>{" "}
          <span className="alert-inner--text">
            <strong>{i18next.t("Splash.Success")}</strong>
            {i18next.t("Splash.SuccessMsg")}
          </span>
        </UncontrolledAlert>
      );
    }
    return (
      <>
        <Container maxWidth="md">
          <div>
            <Jumbotron className="application-jumbotron">
              <h1 className="display-4">Hey,</h1>
              <p className="lead">{i18next.t("Splash.Welcome")}</p>
            </Jumbotron>
          </div>
          {messageBlock}
          <Form role="form" onSubmit={this.onSubmit}>
            <FormGroup>
              <Label>{i18next.t("Splash.Email")}</Label>
              <Input
                type="email"
                name="email"
                required
                placeholder="example@gmail.com"
                value={this.state.email}
                onChange={this.handleInputChange}
              />
            </FormGroup>
            <FormGroup>
              <Label>
                {i18next.t("Splash.Username")}{" "}
                {this.getUsernameVerificationBadge()}
              </Label>
              <Input
                type="text"
                name="username"
                disabled={this.state.verified || this.state.loading}
                required
                placeholder="username"
                value={this.state.username}
                onChange={this.handleInputChange}
              />
            </FormGroup>
            <Card>
              <CardBody>
                {i18next.t("Splash.VefiryInstruction")}
                <ol>
                  <li>{i18next.t("Splash.VerifyStep1")}</li>
                  <div>
                    <QRCode
                      style={{ display: "none" }}
                      id="qrcode"
                      value={this.VERIFICATION_QRCODE_CONTENT}
                      size={32}
                      level={"H"}
                      includeMargin={false}
                    />
                    <Button
                      outline
                      size="sm"
                      color="info"
                      onClick={this.downloadQR}
                    >
                      {" "}
                      {i18next.t("Splash.DownloadButton")}{" "}
                    </Button>
                  </div>
                  <li>
                    {i18next.t("Splash.VerifyStep2-1")}{" "}
                    <a
                      href="https://my.minecraft.net/en-us/profile/skin"
                      target="_blank"
                    >
                      https://my.minecraft.net/en-us/profile/skin
                    </a>
                    , {i18next.t("Splash.VerifyStep2-2")}
                  </li>
                  <li>{i18next.t("Splash.VerifyStep3")}</li>
                  <li>
                    {i18next.t("Splash.VerifyStep4")} <br></br>
                    <Button
                      size="sm"
                      outline
                      style={{ display: this.state.loading ? "none" : "" }}
                      disabled={this.state.verified}
                      onClick={this.onVerifyUsername}
                      color="primary"
                    >
                      {i18next.t("Splash.VerifyButton")}
                    </Button>
                    <Spinner
                      style={{ display: this.state.loading ? "" : "none" }}
                      color="primary"
                    />
                  </li>
                  <li>{i18next.t("Splash.VerifyStep5")}</li>
                </ol>
              </CardBody>
            </Card>
            <FormGroup>
              <Label>{i18next.t("Splash.Gender")}</Label>
              <Input
                type="select"
                name="gender"
                required
                value={this.state.gender}
                onChange={this.handleInputChange}
              >
                <option>{i18next.t("Splash.Gender_Male")}</option>
                <option>{i18next.t("Splash.Gender_Female")}</option>
                <option>{i18next.t("Splash.Gender_Other")} </option>
              </Input>
            </FormGroup>
            <FormGroup>
              <Label>{i18next.t("Splash.Age")}</Label>
              <Input
                type="number"
                name="age"
                required
                value={this.state.age}
                onChange={this.handleInputChange}
              />
            </FormGroup>
            <FormGroup>
              <Label>{i18next.t("Splash.ApplicationText")}</Label>
              <Input
                type="textarea"
                rows="8"
                cols="60"
                minLength="100"
                required
                name="applicationText"
                placeholder={i18next.t("Splash.ApplicationTextPlaceholder")}
                value={this.state.applicationText}
                onChange={this.handleInputChange}
              />
            </FormGroup>
            <FormGroup check>
              <Label check>
                <Input type="checkbox" required />{" "}
                {i18next.t("Splash.RadioboxDescription")}
              </Label>
            </FormGroup>
            <Recaptcha
              ref={ref => (this.recaptcha = ref)}
              sitekey={RECAPTCHA_SITEKEY}
              onResolved={this.onResolved}
            />
            <Button
              disabled={!this.state.verified}
              color="primary"
              type="submit"
              size="lg"
            >
              {i18next.t("Splash.SubmitButton")}
            </Button>
          </Form>
        </Container>
      </>
    );
  }
}

export default Application;
