import React from "react";
import { Button, Form, FormGroup, Label, Input, FormText, Container } from 'reactstrap';
import axios from "axios";

class Application extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      email : '',
      username: '',
      gender: '',
      age: 0,
      applicationText: ''
    };
  }
  onSubmit = (event) => {
    let api_base_url = "http://localhost:8080"
    event.preventDefault();
    console.log(this.state)
    axios.post(`${api_base_url}/api/v1/requests/`, {
        email: this.state.email,
        username: this.state.username,
        gender: this.state.gender,
        age: parseInt(this.state.age),
        applicationText: this.state.applicationText

    })
    .then(res => {
      if (res.status === 200) {
          alert("Successfully submitted. You should receive an email for details")
      }else if(res.status === 500) {
          alert("Internal server error. Please contact admin or try later")
      }else if(res.status === 400) {
          alert(res.data.message)
      }
    })
    .catch(err => {
        console.log(err)
    })
  }

  handleInputChange = (event) => {
    const { value, name } = event.target;
    this.setState({
      [name] : value
    });
  }
    render() {
        return (
        <>
    <Container>
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
        <Input type="number" name="age" required placeholder="" value={this.state.age}  onChange={this.handleInputChange} />
      </FormGroup>
      <FormGroup>
        <Label>Application</Label>
        <Input type="textarea" required name="applicationText" placeholder="tell us about your experience with minecraft and minecraft servers" value={this.state.applicationText}  onChange={this.handleInputChange}/>
      </FormGroup>
      <FormGroup check>
        <Label check>
          <Input type="checkbox" />{' '}
          I've read the server rules and I agree to submit my application to join
        </Label>
      </FormGroup>
      <Button type="submit">Submit Application</Button>
    </Form>
    </Container>
        </>

        );
    }
  }


 export default Application;

