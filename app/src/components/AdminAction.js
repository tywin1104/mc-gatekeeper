import React from "react";
import { ListGroup, ListGroupItem, Container, Button, Jumbotron, Alert, Form, FormGroup, Input } from 'reactstrap';
import moment from 'moment'
import RequestsService from '../service/RequestsService'
import './AdminAction.css';

class AdminAction extends React.Component {
  constructor(props) {
    super()
    this.state = {
        currentRequest: {},
        invalid: false,
        adminToken: "",
        note: "",
    };
  }
  componentDidMount() {
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
    const { match: { params } } = this.props;
    RequestsService.verifyAdminToken(params.id, adminToken)
    .catch(error => {
        this.setState({
            invalid: true
        })
        return
    });
    // Get current request related to this specific email ticket?
    RequestsService.getRequestByEncodedID(params.id)
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

  handleInputChange = (event) => {
    const { value, name } = event.target;
    this.setState({
      [name] : value
    });
  }

  onDecision = (event, newStatus) => {
    event.preventDefault();
    const { match: { params } } = this.props;
    let note = this.state.note
    let promise
    if (newStatus === "Denied") {
        promise = RequestsService.denyRequest(params.id, this.state.adminToken, note)
    }else {
        promise = RequestsService.approveRequest(params.id, this.state.adminToken, note)
    }
    promise
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
            <ListGroupItem active  action>Request Summary</ListGroupItem>
            <ListGroupItem  action><strong>Gender</strong> {currentRequest.gender}</ListGroupItem>
            <ListGroupItem  action><strong>Age</strong> {currentRequest.age}</ListGroupItem>
            <ListGroupItem  action>
                <strong>Application Text</strong>
                <Jumbotron>
                    <p>{currentRequest.info.applicationText}</p>
                </Jumbotron>
            </ListGroupItem>
            <ListGroupItem disabled  action>Application submitted { moment.parseZone(currentRequest.timestamp).local().fromNow()}</ListGroupItem>
        </ListGroup>
        <Form>
            <FormGroup>
            <Input type="textarea" name="note" placeholder="Add note here (Optional)" value={this.state.note}  onChange={this.handleInputChange} />
            </FormGroup>
        </Form>
        <Button className="actionButton"
            onClick={(e) => this.onDecision(e, "Approved")}  color="success" outline  size="lg" type="button">
            Approve
        </Button>
        <Button className="actionButton"
            onClick={(e) => this.onDecision(e, "Denied")} color="danger" outline  size="lg" type="button">
            Deny
        </Button>

        <Jumbotron fluid>
            <Container fluid>
            <h3>Notes to ops:</h3>
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

