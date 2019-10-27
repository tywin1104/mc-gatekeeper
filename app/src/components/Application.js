import React from "react";
import { Button, Form, FormGroup, Label, Input, UncontrolledAlert, Jumbotron} from 'reactstrap';
import axios from "axios";
import './Application.css';
import Container from '@material-ui/core/Container';
import { ReCaptcha } from 'react-recaptcha-google'

class Application extends React.Component {
  constructor(props, context) {
    super(props, context);
    this.onLoadRecaptcha = this.onLoadRecaptcha.bind(this);
    this.verifyCallback = this.verifyCallback.bind(this);
    this.state = {
      email : '',
      username: '',
      gender: '',
      age: '',
      applicationText: '',
      errorMsg: '',
      success: false,
      recaptchaVerified: false
    };
  }
  componentDidMount() {
    if (this.submitionCaptcha) {
        console.log("started, just a second...")
        this.submitionCaptcha.reset();
    }
  }
  onLoadRecaptcha() {
      if (this.submitionCaptcha) {
          this.submitionCaptcha.reset();
      }
  }
  verifyCallback(recaptchaToken) {
    this.setState({recaptchaVerified: true})
    // console.log(recaptchaToken, "<= your recaptcha token")
  }

  onSubmit = (event) => {
    let api_base_url = process.env.REACT_APP_API_BASE_URL
    event.preventDefault();
    axios.post(`${api_base_url}/api/v1/requests/`, {
        email: this.state.email,
        username: this.state.username,
        gender: this.state.gender,
        age: parseInt(this.state.age),
        info: {
          applicationText: this.state.applicationText
        }
    })
    .then(res => {
      if (res.status === 200) {
          this.setState({
              success: true
          })
      }})
    .catch(error => {
        if (error.response) {
            if(error.response.status === 400) {
                this.setState({
                    errorMsg: error.response.data.message
                })
            }else {
                this.setState({
                    errorMsg: "Internal server error. Please try later or contact server admin"
                })
            }
        }
    });
  }

  handleInputChange = (event) => {
    const { value, name } = event.target;
    this.setState({
      [name] : value
    });
  }
render() {
    let messageBlock
    if(this.state.errorMsg) {
        messageBlock = (
            <UncontrolledAlert color="danger" fade={false}>
            <span className="alert-inner--icon">
                <i className="ni ni-like-2" />
            </span>{" "}
            <span className="alert-inner--text">
                <strong>Error</strong> { this.state.errorMsg }
            </span>
            </UncontrolledAlert>
        )
    }else if(this.state.success) {
        messageBlock = (
            <UncontrolledAlert color="success" fade={false}>
            <span className="alert-inner--icon">
                <i className="ni ni-like-2" />
            </span>{" "}
            <span className="alert-inner--text">
                <strong>Success</strong> Your application is on the way.. Check the email for confirmation.
            </span>
            </UncontrolledAlert>
        )
    }
    return (
        <>
    <Container maxWidth="md">
    <div>
    <Jumbotron className="jumbotron">
        <h1 className="display-4">Hey,</h1>
        <p className="lead">Please kindly fill in the form for request to join our server. Our server admin will handle the applications within 24 hours. See you there! </p>
      </Jumbotron>
    </div>
    { messageBlock }
    <Form role="form" onSubmit={this.onSubmit}>
      <FormGroup>
        <Label>Email</Label>
        <Input type="email" name="email"  required placeholder="your email address"  value={this.state.email}  onChange={this.handleInputChange} />
      </FormGroup>
      <FormGroup>
        <Label >Minecraft ID (username)</Label>
        <Input type="text" name="username" required  placeholder="minecraft username"  value={this.state.username}  onChange={this.handleInputChange}/>
      </FormGroup>
      <FormGroup>
        <Label>Gender</Label>
        <Input type="select" name="gender" value={this.state.gender}  onChange={this.handleInputChange}>
          <option>male</option>
          <option>female</option>
          <option>Other</option>
        </Input>
      </FormGroup>
      <FormGroup>
        <Label >Age</Label>
        <Input type="number" name="age" required value={this.state.age}  onChange={this.handleInputChange} />
      </FormGroup>
      <FormGroup>
        <Label>Application</Label>
        <Input type="textarea"  rows='8' cols='60' required name="applicationText" placeholder="tell us about your experience with minecraft and minecraft servers" value={this.state.applicationText}  onChange={this.handleInputChange}/>
      </FormGroup>
      <FormGroup check>
        <Label check>
          <Input type="checkbox" required/>{' '}
          I've read the <a href="#">server rules</a> and I agree to submit my application to join
        </Label>
      </FormGroup>
      <ReCaptcha
            ref={(el) => {this.submitionCaptcha = el;}}
            size="normal"
            data-theme="dark"
            render="explicit"
            sitekey="6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI"
            onloadCallback={this.onLoadRecaptcha}
            verifyCallback={this.verifyCallback}
        />
      <Button color="primary" disabled={!this.state.recaptchaVerified} type="submit"  size="lg">
        Submit Application
      </Button>
    </Form>
    </Container>
        </>
        );
    }
  }


 export default Application;

