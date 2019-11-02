import React from 'react';
import TextField from '@material-ui/core/TextField';
import Button from '@material-ui/core/Button';
import Typography from '@material-ui/core/Typography';
import Container from '@material-ui/core/Container';
import AuthService from '../service/AuthService';
import Recaptcha from 'react-google-invisible-recaptcha';
import RecaptchaService from '../service/RecaptchaService'

const RECAPTCHA_SITEKEY = window.RECAPTCHA_SITEKEY ? window.RECAPTCHA_SITEKEY : "";
console.log(RECAPTCHA_SITEKEY)

class Login extends React.Component {

    constructor(props, context){
        super(props, context);
        this.state = {
            username: '',
            password: '',
        }
        this.login = this.login.bind(this);
        this.onResolved = this.onResolved.bind( this );
    }

    componentDidMount() {
        localStorage.clear();
    }



    login = (e) => {
        e.preventDefault();
        this.recaptcha.reset()
        this.recaptcha.execute();
    };

    onResolved() {
        RecaptchaService.verify(this.recaptcha.getResponse())
        .then(res => {
            if (res.status === 200 && res.data.success) {
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
            }
        })
      }

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
                    <Recaptcha
                    ref={ ref => this.recaptcha = ref }
                    sitekey={RECAPTCHA_SITEKEY}
                    onResolved={ this.onResolved } />
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