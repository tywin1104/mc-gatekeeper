import React from "react";
import { Button, Form, FormGroup, Label, Input, UncontrolledAlert, Jumbotron} from 'reactstrap';
import './Application.css';
import Container from '@material-ui/core/Container';
import Recaptcha from 'react-google-invisible-recaptcha';
import RequestsService from '../service/RequestsService'
import RecaptchaService from '../service/RecaptchaService'
import i18next from "i18next";

const RECAPTCHA_SITEKEY = window.RECAPTCHA_SITEKEY ? window.RECAPTCHA_SITEKEY : process.env.REACT_APP_RECAPTCHA_SITEKEY;
console.log(RECAPTCHA_SITEKEY)
class Application extends React.Component {
  constructor(props) {
    super(props);
    this.onResolved = this.onResolved.bind( this );
    this.state = {
      email : '',
      username: '',
      gender: 'male',
      age: '',
      applicationText: '',
      errorMsg: '',
      success: false,
    };
  }

  onResolved() {
    RecaptchaService.verify(this.recaptcha.getResponse())
    .then(res => {
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
    }})
  }

  onSubmit = (event) => {
    event.preventDefault();
    this.recaptcha.reset();
    this.recaptcha.execute();
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
                <strong>Success</strong> Your application is on the way.. Check your email for confirmation.
            </span>
            </UncontrolledAlert>
        )
    }
    return (
        <>
    <Container maxWidth="md">
    <div>
    <Jumbotron className="application-jumbotron">
        <h1 className="display-4">Hey,</h1>
        {/* <p className="lead">Please kindly fill in the form for request to join our server. Our server admin will handle the applications within 24 hours. See you there! </p> */}
        <p className="lead">{i18next.t('Splash.Welcome')}</p>
      </Jumbotron>
    </div>
    { messageBlock }
    <Form role="form" onSubmit={this.onSubmit}>
      <FormGroup>
        <Label>{i18next.t('Splash.Email')}</Label>
        <Input type="email" name="email"  required placeholder="your email address"  value={this.state.email}  onChange={this.handleInputChange} />
      </FormGroup>
      <FormGroup>
        <Label >{i18next.t('Splash.Username')}</Label>
        <Input type="text" name="username" required  placeholder="minecraft username"  value={this.state.username}  onChange={this.handleInputChange}/>
      </FormGroup>
      <FormGroup>
        <Label>{i18next.t('Splash.Gender')}</Label>
        <Input type="select" name="gender" required value={this.state.gender}  onChange={this.handleInputChange}>
          <option>male</option>
          <option>female</option>
          <option>Other</option>
        </Input>
      </FormGroup>
      <FormGroup>
        <Label >{i18next.t('Splash.Age')}</Label>
        <Input type="number" name="age" required value={this.state.age}  onChange={this.handleInputChange} />
      </FormGroup>
      <FormGroup>
        <Label>{i18next.t('Splash.ApplicationText')}</Label>
        <Input type="textarea"  rows='8' cols='60' minLength="100" required name="applicationText" placeholder={i18next.t('Splash.ApplicationTextPlaceholder')} value={this.state.applicationText}  onChange={this.handleInputChange}/>
      </FormGroup>
      <FormGroup check>
        <Label check>
          <Input type="checkbox" required/>{' '}
          {i18next.t('Splash.RadioboxDescription')}
        </Label>
      </FormGroup>
      <Recaptcha
          ref={ ref => this.recaptcha = ref }
          sitekey={RECAPTCHA_SITEKEY}
          onResolved={ this.onResolved } />
      <Button color="primary" type="submit"  size="lg">
        Submit Application
      </Button>
    </Form>
    </Container>
        </>
        );
    }
  }


 export default Application;

