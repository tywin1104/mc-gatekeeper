import React from "react";
import axios from "axios";
import { ListGroup, ListGroupItem, Container, Button, Jumbotron, Alert } from 'reactstrap';
import moment from 'moment'
import './CheckStatus.css';

class AdminAction extends React.Component {
  constructor(props) {
    super()
    this.state = {
        currentRequest: {},
        invalid: false,
        adminToken: ""
    };
  }
  componentDidMount() {
    let api_base_url = process.env.REACT_APP_API_BASE_URL
    const search = this.props.location.search;
    const queryParams = new URLSearchParams(search);
    let adminToken = queryParams.get('adm')
    if(adminToken == null) {
        this.setState({
            invalid: true
        })
        return
    }else {
        this.setState({
            adminToken: adminToken
        })
    }

    // Verify adminToken is valid
    axios.get(`${api_base_url}/api/v1/verify?adm=${adminToken}`)
    .catch(error => {
        this.setState({
            invalid: true
        })
        return
    });
    const { match: { params } } = this.props;
    axios.get(`${api_base_url}/api/v1/requests/${params.id}`)
    .then(res => {
        if(res.status === 200) {
            this.setState({
                currentRequest : res.data.request
            })
        }
    })
    .catch(error => {
        this.setState({
            invalid: true
        })
        return
    });

  }

  onApprove = (event) => {
    let api_base_url = process.env.REACT_APP_API_BASE_URL
    event.preventDefault();
    const { match: { params } } = this.props;
    axios.patch(`${api_base_url}/api/v1/requests/${params.id}?adm=${this.state.adminToken}`, {
        status: "Approved",
        processedTimestamp: new Date(),
        admin: 'admin'
    })
    .then(res => {
      if (res.status === 200) {
          alert("Completed! Thank you!")
          window.location.reload();
      }})
    .catch(error => {
        if (error.response) {
            if(error.response.status === 400) {
                alert("Invalid token. Please do not modify the original link sent to you via email")
            }else {
                alert("Unable to perform action due to internal server error")
            }
        }
    });
  }

onDeny = (event) => {
    let api_base_url = process.env.REACT_APP_API_BASE_URL
    event.preventDefault();
    const { match: { params } } = this.props;
    axios.patch(`${api_base_url}/api/v1/requests/${params.id}?adm=${this.state.adminToken}`, {
        status: "Denied",
        processedTimestamp: new Date(),
        admin: 'admin'
    })
    .then(res => {
      if (res.status === 200) {
          alert("Completed! Thank you!")
          window.location.reload();
      }})
    .catch(error => {
        if (error.response) {
            if(error.response.status === 400) {
                alert("Invalid token. Please do not modify the original link sent to you via email")
            }else {
                alert("Unable to perform action due to internal server error")
            }
        }
    });
  }
  render() {
      let display
      let currentRequest = this.state.currentRequest
      if(!this.state.invalid && currentRequest && this.state.currentRequest.status === "Pending") {
          display = (
        <Container>
        <ListGroup>
            <ListGroupItem active  action>Request Infomation</ListGroupItem>
            {/* <ListGroupItem  action><strong>Minecraft Username</strong> ******</ListGroupItem> */}
            <ListGroupItem  action><strong>Gender</strong> {currentRequest.gender}</ListGroupItem>
            <ListGroupItem  action><strong>Age</strong> {currentRequest.age}</ListGroupItem>
            <ListGroupItem  action>{currentRequest.info.applicationText}</ListGroupItem>
            <ListGroupItem disabled  action>Application submitted { moment.parseZone(currentRequest.timestamp).local().fromNow()}</ListGroupItem>
        </ListGroup>
        <Button onClick={this.onApprove} color="success" outline  size="lg" type="button">
            Approve
        </Button>
        <Button onClick={this.onDeny} color="danger" outline  size="lg" type="button">
            Deny
        </Button>

        <Jumbotron fluid>
            <Container fluid>
            <h3>Creterias for whitelist approval</h3>
            <p className="lead">
            orem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.
            </p>
            </Container>
        </Jumbotron>
        </Container>
        )
      }else if(currentRequest && (currentRequest.status === "Approved" || currentRequest.status === "Denied")) {
        display = (
        <div>
        <Alert color="info">
            The request you are looking at is already fulfilled.
            Thank you for taking your time.
        </Alert>
        </div>
        )
      }else if(this.state.invalid){
        display = (
            <p>Invalid route</p>
        )
      }
      return (
          <div>
              {display}
          </div>
      )
  }
}


 export default AdminAction;

