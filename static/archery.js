window.onload = function(){
	var app = new Vue({
	el: '#app',
	data: {
		target_qps: '',
		qps_step: '',
		status_string: 'stop',
		active_nav_tab: 'start',
		work_list: [],
		my_chart: {},
		my_chart_server: {},
		archery_status: 0, //0:stop 1:start;
		chart_controller: {},
		data_series: {},
		cpu: [],
		memory: [],
		server_date_string: [],
		monitor: false,
		slave_num: -1,
	},
	methods: {
		getMappingValueArrayOfKey(map,keyName){
			sum = 0
			for (var key in map) {
				sum += map[key][keyName]
			}
			return sum
		},
		show_start(){
			this.active_nav_tab = 'start';
			console.log(this.active_nav_tab)
		},
		show_chart(){
			this.active_nav_tab = 'chart';
		},
		show_sum(){
			this.active_nav_tab = 'sum';
		},
		stop_test(){
			axios.post("stop","stop").then(() => {
				setTimeout(() => {
					console.log(this.chart_controller)
					clearInterval(this.chart_controller);
					this.status_string = 'stop';
					this.active_nav_tab = 'start';
					this.archery_status = 0
				},1000);
			});
		},
		start_test(){
			if (this.archery_status == 1) {
				alert("请先停下当前测试任务，再开始新的任务");
				return
			}
			axios.post("start",{"target-qps":Number(this.target_qps),"increase-per-second":Number(this.qps_step)}).then((response) => {
				this.archery_status = 1;
				this.chart_controller = setInterval(this.get_test_data, 1000);
				this.show_chart();
				console.log(this.chart_controller)
		})
		},
		get_test_data(){
			axios.get("get_second_data").then((response) => {
				for (key in response.data.one_second_data_obj) {
					this.data_series[key].req_num.push(response.data['one_second_data_obj'][key]['request_num'])
					this.data_series[key].succ_resp_num.push(response.data['one_second_data_obj'][key]['success_response_num'])
					this.data_series[key].fail_num.push(response.data['one_second_data_obj'][key]['failed_num'])
					this.data_series[key].delay.push(response.data['one_second_data_obj'][key]['average_cost_time'])
					this.data_series[key].delay_99.push(response.data['one_second_data_obj'][key]['ninty_nine_percent_cost_time'])
					this.data_series[key].delay_90.push(response.data['one_second_data_obj'][key]['ninty_percent_cost_time'])
					this.data_series[key].delay_middle.push(response.data['one_second_data_obj'][key]['fifty_percent_cost_time'])
					time_stamp = response.data['one_second_data_obj'][key]['time_stamp']
					var date = new Date(time_stamp*1000);
					Y = date.getFullYear() + '-';
					M = (date.getMonth()+1 < 10 ? '0'+(date.getMonth()+1) : date.getMonth()+1) + '-';
					D = date.getDate() + ' ';
					h = date.getHours() + ':';
					m = date.getMinutes() + ':';
					s = date.getSeconds();
					this.data_series[key].date_string.push(Y+M+D+h+m+s)
					opt = {
						xAxis:{
							data:this.data_series[key].date_string
						},
						
						series:[{
								name:'请求数',
								data:this.data_series[key].req_num
							},
							{
								name:'成功响应数',
								data:this.data_series[key].succ_resp_num
							},
							{
								name:'失败数',
								data:this.data_series[key].fail_num
							},
							{
								name:'平均响应延时',
								yAxisIndex:1,
								data:this.data_series[key].delay
							},
							{
								name:'99%响应延时',
								yAxisIndex:1,
								data:this.data_series[key].delay_99
							},
							{
								name:'90%响应延时',
								yAxisIndex:1,
								data:this.data_series[key].delay_90
							},
							{
								name:'响应延时中位数',
								yAxisIndex:1,
								data:this.data_series[key].delay_middle
							},
						]
					}
					console.log(Object.keys(this.my_chart))
					this.my_chart[key].setOption(opt);
					if(Object.keys(response.data.one_second_data_obj).length == 1){
						this.status_string = "Running: " + response.data['one_second_data_obj'][Object.keys(response.data['one_second_data_obj'])[0]]['request_num'] + " requests/sec"
					} else {
						this.status_string = Object.keys(response.data.one_second_data_obj).length + " Work " + "Running: " + this.getMappingValueArrayOfKey(response.data.one_second_data_obj, 'request_num') + " requests/sec"
					}
					this.data_series[key].req_num_sum = response.data['one_second_data_obj'][key]['total_request_num']
					this.data_series[key].succ_resp_num_sum = response.data['one_second_data_obj'][key]['total_succ_response_num']
					this.data_series[key].failed_num_sum = response.data['one_second_data_obj'][key]['total_failed_num']
					this.data_series[key].succ_resp_average_cost = response.data['one_second_data_obj'][key]['total_succ_resp_time'] / response.data['one_second_data_obj'][key]['total_succ_response_num']
				}
				
				//console.log(opt);
				
				if(response.data['target_server_data']['time_stamp'] != 0){
					this.cpu.push(response.data['target_server_data']['cpu_rate']/100.0)
					this.memory.push(response.data['target_server_data']['memory_usage']/100.0)
					server_time_stamp = response.data['target_server_data']['time_stamp']
					var date = new Date(server_time_stamp*1000);
					Y = date.getFullYear() + '-';
					M = (date.getMonth()+1 < 10 ? '0'+(date.getMonth()+1) : date.getMonth()+1) + '-';
					D = date.getDate() + ' ';
					h = date.getHours() + ':';
					m = date.getMinutes() + ':';
					s = date.getSeconds();
					this.server_date_string.push(Y+M+D+h+m+s)
					opt = {
						xAxis:{
							data:this.server_date_string
						},
						series:[{
							name:'CPU占用',
							data:this.cpu
						},
							{
								name:'内存占用',
								data:this.memory
							},
						]
					}
					this.my_chart_server.setOption(opt);
				}
				// $("#req_num_sum").text(JSON.parse(data)['test_data_sum']['total_request_num'])
				// $("#succ_resp_num_sum").text(JSON.parse(data)['test_data_sum']['total_succ_response_num'])
				// var average_time = (JSON.parse(data)['test_data_sum']['total_succ_resp_time']) / JSON.parse(data)['test_data_sum']['total_succ_response_num']
				// $("#succ_resp_average_cost").text(average_time.toFixed(3))
				// $("#failed_num_sum").text(JSON.parse(data)['test_data_sum']['total_failed_num'])
				// server_status = JSON.parse(data)['server_status']
				// if (server_status == 0) {
				// 	this.status_string = 'Stop'
				// }
				// else if (server_status == 1){
				// 	this.status_string = "Running: " + response.data['one_second_data_obj'][key]['request_num'] + " requests/sec"
				// }
				// slave_num = JSON.parse(data)['slave_num']
				// if (slave_num > 0){
				// 	$("#slave").text(slave_num +' Slaves');
				// }
			});
		},
		init_chart(id){
			// console.log(id)
			this.my_chart[id] = echarts.init(document.getElementById(id));
			this.my_chart[id].setOption({
				title:{
					text:'请求及延时 - ' + id,
					x:'center'
				},
				tooltip:{
					trigger: 'axis'
				},
				legend:{
					top: "10%",
					data:['请求数','成功响应数','失败数','平均响应延时','99%响应延时','90%响应延时','响应延时中位数']
				},
				toolbox: {
					show: true,
					orient: 'vertical',
					top: 10,
					right: "5%",
					feature: {
						saveAsImage: {
							show: true,
							type:'jpg',
							name: id,
							title:'保存为图片',
						},
						dataView: {
							show: true,
							title: '数据视图',
							readOnly: false,
						}
					},
					
				},
				xAxis:{
					boundaryGap: false,
					data:[]
				},
				yAxis:[
					{
						name: "请求/响应数",
						type: "value"
					},
					{
						name: "响应延时",
						type: "value"
					}
				],
				
			series:[{
				name:'请求数',
				type:'line',
				data:[]
				},
				{
				name:'成功响应数',
				type:'line',
				data:[]
				},
				{
				name:'失败数',
				type:'line',
				data:[]
				},
				{
				name:'平均响应延时',
				type:'line',
				yAxisIndex:1,
				data:[]
				},
				{
				name:'99%响应延时',
				type:'line',
				yAxisIndex:1,
				data:[]
				},
				{
				name:'90%响应延时',
				type:'line',
				yAxisIndex:1,
				data:[]
				},
				{
				name:'响应延时中位数',
				type:'line',
				yAxisIndex:1,
				data:[]
				},
			]
		});

		// $.get("get_second_data",function(data,status){
		// 	server_status = JSON.parse(data)['server_status']

		// 	slave_num = JSON.parse(data)['slave_num']
		// 	if (slave_num > 0){
		// 		$("#slave").text(slave_num +' Slaves');
		// 	}
			// if (JSON.parse(data)['target_server_data']['time_stamp'] != 0){
			// 	$("#chart_server").show();
			// }
			// else{
			// 	$("#chart_server").hide();
			// }
		// });
		}
	},
	mounted: function(){
		this.show_start()
		axios.get('/get_second_data').then((response) => {
			tmp_data_series = {}
			for (key in response.data.one_second_data_obj) {
				tmp_data_series[key] = {
					req_num: [],
					succ_resp_num: [],
					delay: [],
					delay_99: [],
					delay_90: [],
					delay_middle: [],
					fail_num: [],
					date_string: [],
					failed_num_sum: 0,
					succ_resp_num_sum: 0,
					req_num_sum: 0,
					succ_resp_average_cost: 0
				}
				this.data_series = tmp_data_series
				let tmp = key
				setTimeout(() => {
					console.log(tmp)
					this.init_chart(tmp)
				}, 100);
			}
			this.my_chart_server = echarts.init(document.getElementById('chart_server'));
			this.my_chart_server.setOption({
				title:{
					text:'目标服务器资源'
				},
				tooltip:{
					trigger: 'axis'
				},
				legend:{
					data:['CPU占用','内存占用']
				},
				xAxis:{
					boundaryGap: false,
					data:[]
				},
				yAxis:[
					{
						name: "CPU占用(100%-idle)",
						type: "value"
					},
					{
						name: "内存占用(100%-avaiable)",
						type: "value"
					}
				],
				series:[{
					name:'CPU占用',
					type:'line',
					data:[]
					},
					{
					name:'内存占用',
					type:'line',
					yAxisIndex:1,
					data:[]
					}
				]
			});
			if (response.data.server_status == 0) {
				this.status_string = 'Stop'
			}
			else if (response.data.server_status == 1){
				if(Object.keys(response.data.one_second_data_obj).length == 1){
					this.status_string = "Running: " + response.data['one_second_data_obj'][Object.keys(response.data['one_second_data_obj'])[0]]['request_num'] + " requests/sec"
				} else {
					this.status_string = "Running: " + Object.keys(response.data.one_second_data_obj).length + " work"
				}
				chart_controller = setInterval(this.get_test_data, 1000);
				this.show_chart();
				this.archery_status = 1;
			}
			if (response.data['target_server_data']['time_stamp'] != 0){
				this.monitor = true
			}
			else{
				this.monitor = false
			}
			this.slave_num = response.data['slave_num']
		})
	}
	})
}