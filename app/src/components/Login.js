import React from 'react';
import TextField from '@material-ui/core/TextField';
import Button from '@material-ui/core/Button';
import Typography from '@material-ui/core/Typography';
import Container from '@material-ui/core/Container';
import AuthService from '../service/AuthService';
import { ReCaptcha } from 'react-recaptcha-google'

class Login extends React.Component {

    constructor(props, context){
        super(props, context);
        this.state = {
            username: '',
            password: '',
        }
        this.login = this.login.bind(this);
        this.onLoadRecaptcha = this.onLoadRecaptcha.bind(this);
        this.verifyCallback = this.verifyCallback.bind(this);
    }

    componentDidMount() {
        localStorage.clear();
        if (this.captchaDemo) {
            console.log("started, just a second...")
            this.captchaDemo.reset();
            this.captchaDemo.execute();
        }
      }

      onLoadRecaptcha() {
          if (this.captchaDemo) {
              this.captchaDemo.reset();
              this.captchaDemo.execute();
          }
      }

      verifyCallback(recaptchaToken) {
        // console.log(recaptchaToken, "<= your recaptcha token")
        this.login()
      }

    login = (e) => {
        e.preventDefault();
        const credentials = {username: this.state.username, password: this.state.password};
        AuthService.login(credentials).then(res => {
            if(res.status === 200){
                localStorage.setItem("token", JSON.stringify(res.data.token));
                this.props.history.push('/dashboard');
            }
        })
        .catch(error => {
            if (error.response) {
                if(error.response.status === 401) {
                    alert("Wrong credentials")
                    this.setState({
                        // clear input field
                        username: "",
                        password: ""
                    })
                }
            }
        });
    };

    onChange = (e) =>
        this.setState({ [e.target.name]: e.target.value });

    render() {
        return(
            <React.Fragment>
                <Container maxWidth="sm">
                    <Typography variant="h4" style={styles.center}>Login</Typography>
                    <form onSubmit={this.login}>
                        <Typography variant="h4" style={styles.notification}>{this.state.message}</Typography>
                        <TextField type="text" label="USERNAME" fullWidth margin="normal" name="username" value={this.state.username} onChange={this.onChange} required/>

                        <TextField type="password" label="PASSWORD" fullWidth margin="normal" name="password" value={this.state.password} onChange={this.onChange} required/>

                        <Button variant="contained" color="secondary" type="submit">Login</Button>
                    </form>
                    <ReCaptcha
                        ref={(el) => {this.captchaDemo = el;}}
                        size="invisible"
                        render="explicit"
                        sitekey="6Lc_vL8UAAAAAMNIAhLWtEDyoQDtjzwygxP1knim"
                        onloadCallback={this.onLoadRecaptcha}
                        verifyCallback={this.verifyCallback}
                    />
                </Container>
            </React.Fragment>
        )
    }
}

const styles= {
    center :{
        display: 'flex',
        justifyContent: 'center'

    },
    notification: {
        display: 'flex',
        justifyContent: 'center',
        color: '#dc3545'
    }
}

export default Login;