import React from "react";
import MaterialTable from "material-table";
import List from '@material-ui/core/List';
import ListItem from '@material-ui/core/ListItem';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import ListItemText from '@material-ui/core/ListItemText';
import NotesIcon from '@material-ui/icons/Notes';
import WcIcon from '@material-ui/icons/Wc';
import FaceIcon from '@material-ui/icons/Face';
import CommentIcon from '@material-ui/icons/Comment';
import moment from 'moment'
import RequestsService from '../../service/RequestsService';
import i18next from "i18next";

class Table extends React.Component {
  onApprove = (event, request) => {
    let requestID = request._id
    event.preventDefault();
    RequestsService.approveRequestAdmin(requestID, this.props.config)
    .then(res => {
      if (res.status === 200) {
        this.props.handleChangeRequestStatus(requestID, "Approved")
      }})
    .catch(error => {
       alert(i18next.t('Dashboard.Table.OperationErrMsg'))
    });
  }

  onDeny = (event, request) => {
    let requestID = request._id
    event.preventDefault();
    RequestsService.denyRequestAdmin(requestID, this.props.config)
    .then(res => {
      if (res.status === 200) {
        this.props.handleChangeRequestStatus(requestID, "Denied")
      }})
    .catch(error => {
       alert(i18next.t('Dashboard.Table.OperationErrMsg'))
    });
  }

  render() {
    if(!this.props || this.props.config == null) {
      return <div>Loading...</div>
    }
    // Create a clone of props so that modifications here will not bubbled up
    let requests = JSON.parse(JSON.stringify(this.props.requests))
    // Transform timestamp and null value to be human-readable
    requests = requests.map(function(item){
      item.timestamp =  moment.parseZone(item.timestamp).local().format("MM/DD/YYYY HH:mm")
      // In Golang, time.Time zero value corresponds to a certain timestamp in mongodb
      if(item.processedTimestamp === "0001-01-01T00:00:00Z") {
        item.processedTimestamp = "N/A"
      }else {
        item.processedTimestamp =  moment.parseZone(item.processedTimestamp).local().format("MM/DD/YYYY HH:mm")
      }
      item.admin = item.admin || "N/A"
      return item;
    });
    return (
      <MaterialTable
        title={i18next.t('Dashboard.Table.AllRequests')}
        columns={[
          { title: i18next.t('Dashboard.Table.ID'), field: '_id' },
          { title: i18next.t('Dashboard.Table.Username'), field: 'username' },
          { title: i18next.t('Dashboard.Table.Email'), field: 'email' },
          { title: i18next.t('Dashboard.Table.Submitted'), field: 'timestamp', },
          { title: i18next.t('Dashboard.Table.Status'), field: 'status',},
          { title: i18next.t('Dashboard.Table.Processed'), field: 'processedTimestamp',},
          { title: i18next.t('Dashboard.Table.Admin'), field: 'admin',},
          { title: i18next.t('Dashboard.Table.Assignees'), field: 'assignees',},
        ]}
        data = {requests}
        detailPanel={rowData => {
          return (
            <div>
               <List component="nav" aria-label="main mailbox folders">
              <ListItem button>
                <ListItemIcon>
                  <WcIcon />
                </ListItemIcon>
                <ListItemText primary={rowData.gender} />
              </ListItem>
              <ListItem button>
                <ListItemIcon>
                  <FaceIcon />
                </ListItemIcon>
                <ListItemText primary={rowData.age} />
              </ListItem>
              <ListItem button>
                <ListItemIcon>
                  <NotesIcon /> <stong>{i18next.t('Dashboard.Table.ApplicationText')}</stong>
                </ListItemIcon>
                <ListItemText primary={rowData.info.applicationText}/>
              </ListItem>
              {/* Hide note section if the data does not contain it */}
              <ListItem button style={{display: rowData.note ? '' : 'none' }}>
                <ListItemIcon>
                  <CommentIcon /> <stong>{i18next.t('Dashboard.Table.Note')}</stong>
                </ListItemIcon>
                <ListItemText primary={rowData.note}/>
              </ListItem>
            </List>
            </div>
          )
        }}
        onRowClick={(event, rowData, togglePanel) => togglePanel()}
        actions={[
          rowData => ({
            icon: 'check',
            tooltip: i18next.t('Dashboard.Table.ApproveTooltip'),
            onClick: (event, rowData) => this.onApprove(event, rowData),
            disabled: rowData.status !== "Pending"
          }),
          rowData => ({
            icon: 'close',
            tooltip: i18next.t('Dashboard.Table.DenyTooltip'),
            onClick: (event, rowData) => this.onDeny(event, rowData),
            disabled: rowData.status !== "Pending"
          })
        ]}
        options={{
          actionsColumnIndex: -1,
          exportButton: true,
          grouping: true
        }}
      />
    )
  }
}

export default Table;